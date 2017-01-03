package main

import (
	"fmt"
	"log"

	"google.golang.org/api/drive/v3"
)

type albums struct {
	root   string
	albums []*drive.File
	srv    *drive.Service
}

func newAlbums(srv *drive.Service) *albums {
	id := getId(srv, "root", "Google Photos", "Albums")
	r, err := srv.Files.List().
		PageSize(1000).
		Fields("files(id,name)").
		Q(fmt.Sprintf(`"%s" in parents and mimeType = "application/vnd.google-apps.folder"`, id)).
		Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}
	return &albums{
		root:   id,
		albums: r.Files,
		srv:    srv,
	}
}

// read retrieves all the files insides a folder and returns a map from hash
// from the md5 to the file ID.
func (a *albums) read(folderID string) map[string]string {
	token := ""
	m := make(map[string]string)
	for {
		r, err := a.srv.Files.List().
			PageSize(1000).
			PageToken(token).
			Spaces("photos").
			Fields("nextPageToken, files(id,md5Checksum)").
			Q(fmt.Sprintf(`mimeType = "image/jpeg" and trashed = false and %q in parents`, folderID)).
			Do()
		if err != nil {
			log.Fatalf("Unable to retrieve files: %v", err)
		}

		if len(r.Files) > 0 {
			for _, f := range r.Files {
				m[f.Md5Checksum] = f.Id
			}
		}

		if r.NextPageToken == "" {
			break
		}
		token = r.NextPageToken
	}
	return m
}

// get looks up an album by name in "Google Photos/Albums/" and returns its ID
// and a map from MD5 hashes to file IDs inside the album. The album will be
// created if it doesn't exist.
func (a *albums) get(albumName string) (ID string, files map[string]string) {
	for _, e := range a.albums {
		if e.Name == albumName {
			return e.Id, a.read(e.Id)
		}
	}
	r, err := a.srv.Files.Create(&drive.File{
		Name:     albumName,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{a.root},
	}).Do()
	log.Printf("Create %q: %+v %v", albumName, r, err)
	if err != nil {
		log.Fatalf("Unable to create folder %q: %v", albumName, err)
	}
	return r.Id, map[string]string{}
}
