package views

import "path"

// icon returns an icon for a file type
func icon(fileName string) string {
	e := path.Ext(fileName)

	switch e {
	case ".tgz":
		return "archive"
	case ".png", ".jpg", ".gif", ".svg":
		return "photo"
	case ".mp3":
		return "music_note"
	}

	return "insert_drive_file"
}
