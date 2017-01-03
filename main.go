// This program syncs local folders with photos with the
// "Google Photos/Albums/" folder from Google Drive.
package main

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

func getDriveService(ctx context.Context) *drive.Service {
	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/album-sync.json
	config, err := google.ConfigFromJSON(b, drive.DriveScope, drive.DrivePhotosReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config)

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
	}
	return srv
}

// getId walks a path indicated by elem and returns the Id of the last
// element. At each step only a single result is expected.
func getId(srv *drive.Service, elem ...string) string {
	parent := ""
	id := ""
	for _, e := range elem {
		if parent == "" {
			parent = e
			continue
		}
		q := fmt.Sprintf(`%q in parents and mimeType = "application/vnd.google-apps.folder" and name = %q`, parent, e)
		r, err := srv.Files.List().Q(q).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve files (%q): %v", q, err)
		}
		if got, want := len(r.Files), 1; got != want {
			log.Fatalf("Unexpected number of results: got %d, want %d", got, want)
		}
		id = r.Files[0].Id
		log.Printf("getId: %q / %q folder: %v\n", parent, e, id)
		parent = id
	}
	return id
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ctx := context.Background()
	srv := getDriveService(ctx)
	cache := newCache(cacheFile())
	cache.update(srv)
	albums := newAlbums(srv)
	for _, a := range albums.albums {
		log.Printf("Existing album: %s %q", a.Id, a.Name)
	}

	for _, dir := range os.Args[1:] {
		log.Printf("Processing %q ...", dir)
		albumID, albumFiles := albums.get(filepath.Base(dir))

		f, err := os.Open(dir)
		if err != nil {
			log.Fatalf("Unable to open dir: %v", err)
		}
		fileInfos, err := f.Readdir(0)
		if err != nil {
			log.Fatalf("Unable to read dir: %v", err)
		}
		for _, fi := range fileInfos {
			if fi.IsDir() {
				continue
			}
			fullPath := filepath.Join(dir, fi.Name())
			data, err := ioutil.ReadFile(fullPath)
			if err != nil {
				log.Printf("Skip %q due to a read error: %v", fullPath, err)
			}
			md5Hash := fmt.Sprintf("%x", md5.Sum(data))
			if _, ok := albumFiles[md5Hash]; ok {
				log.Printf("Skip %q because it already exists", fullPath)
				continue
			}
			f, ok := cache.m[md5Hash]
			if ok {
				r, err := srv.Files.Update(f.ID, nil).AddParents(albumID).Do()
				log.Printf("Adding %q to %s: %v %v", fullPath, albumID, r, err)
			} else {
				fmt.Printf("%q: %s not found\n", fullPath, md5Hash)
			}
		}
	}
}
