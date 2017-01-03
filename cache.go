package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"google.golang.org/api/drive/v3"
)

type driveFile struct {
	Name        string `json:"Name"`
	ID          string `json:"id"`
	Size        int64  `json:"size,string"`
	MD5         string `json:"md5Checksum"`
	CreatedTime string `json:"createdTime"`
}

func cacheFile() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".cache", "album-sync")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir, "files.json")
}

type cache struct {
	m                 map[string]driveFile
	filename          string
	latestCreatedTime string
}

func newCache(filename string) *cache {
	c := cache{
		m:        make(map[string]driveFile),
		filename: filename,
	}
	data, err := ioutil.ReadFile(c.filename)
	if os.IsNotExist(err) {
		return &c
	}
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	json.Unmarshal(data, &c.m)
	for _, f := range c.m {
		if c.latestCreatedTime < f.CreatedTime {
			c.latestCreatedTime = f.CreatedTime
		}
	}
	return &c
}

func (c *cache) save() {
	data, err := json.Marshal(c.m)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	if err := ioutil.WriteFile(c.filename, data, os.ModePerm); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func (c *cache) add(f driveFile) {
	c.m[f.MD5] = f
}

func (c *cache) update(srv *drive.Service) {
	token := ""
	count := 0
	latestCreatedTime := ""

	for {
		r, err := srv.Files.List().
			PageSize(1000).
			PageToken(token).
			Spaces("photos").
			OrderBy("createdTime desc").
			Fields("nextPageToken, files(id,mimeType,name,size,md5Checksum,parents,imageMediaMetadata,createdTime)").
			Q(`mimeType = "image/jpeg" and trashed = false`).
			Do()
		if err != nil {
			log.Fatalf("Unable to retrieve files: %v", err)
		}

		if latestCreatedTime == "" {
			latestCreatedTime = r.Files[0].CreatedTime
		}

		if len(r.Files) > 0 {
			for _, f := range r.Files {
				if f.CreatedTime < c.latestCreatedTime {
					// This early exit assumes that pictures are usually added in
					// incremental CreatedTime. If that's not true then the cache
					// will have to be deleted.
					goto Done
				}
				log.Printf("%-6d %s %s %s %s %q\n", count, f.CreatedTime, f.Id, f.MimeType, f.Md5Checksum, f.Name)
				c.add(driveFile{
					Name:        f.Name,
					ID:          f.Id,
					Size:        f.Size,
					MD5:         f.Md5Checksum,
					CreatedTime: f.CreatedTime,
				})
				count += 1
			}
		}

		if r.NextPageToken == "" {
			break
		}
		token = r.NextPageToken
	}
Done:
	c.latestCreatedTime = latestCreatedTime
	c.save()
}
