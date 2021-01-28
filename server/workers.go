package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

func (s Server) taskPlaylistSync() {
	if s.playlistSyncURL == "" {
		return
	}

	log.Println("PlaylistSync task triggered")
	if err := s.downloader.DownloadPlaylist(s.playlistSyncURL); err != nil {
		log.Println("Error downloading playlist:", err)
	}
}

// taskLimitSize limits the disk size of downloaded videos by deleting
// videos until the downloads directory size is within the specified size limit.
// Videos are deleted in strict reverse chronological order (i.e. taskLimitSize
// will not attempt to delete larger videos before smaller ones).
func (s Server) taskLimitSize() {
	log.Println("taskLimitSize task triggered")

	for {
		fmt.Println("listening")
		j, more := <-s.downloadFinish
		if more {
			fmt.Println("received job", j)

			if s.sizeLimit == 0 {
				continue
			}

			// TODO: we dont want recursive actually --
			// can we use the same logic as GetVideosList
			// and compute sizes from just the videos themselves
			size, err := DirSize(s.downloadsDir)
			if err != nil {
				log.Println("Failed to get directory size:", err)
				continue
			}

			sizeDiff := size - int64(s.sizeLimit)
			if sizeDiff <= 0 {
				continue
			}

			log.Println("deleteing stuff")
			// TODO:
			// - share getVideoFiles across workers and videos list
			// - send to channel on download done
			// - tests
			// - get rid of DirSize

			videos, err := getVideoFiles(s.downloadsDir, createdAsc)
			if err != nil {
				log.Println("Failed to get video files:", err)
				continue
			}

			var toReclaim int64
			toDelete := make([]videoFile, 0, 2)
			for _, v := range videos {
				if toReclaim >= sizeDiff {
					break
				}

				toReclaim += v.Size
				toDelete = append(toDelete, v)
			}

			log.Printf("Deleting %d files to free %d MiB", len(toDelete), toReclaim)
			log.Printf("%v", toDelete)

			// print deleting 10 files to free y MiB
			// for each in todelete: delete
			err = deleteVideoFiles(toDelete)
			if err != nil {
				log.Println("Failed to delete videos:", err)
			}

		} else {
			fmt.Println("received all jobs")
			return
		}
	}
}

// DirSize returns a directory's size in MiB
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size / toMiB, err
}

func (s Server) startWorkers() error {

	// Startup tasks
	if s.playlistSyncURL != "" {
		log.Printf("Tracking playlist: %s\n", s.playlistSyncURL)
		s.taskPlaylistSync()
	}

	size, err := DirSize(s.downloadsDir)
	if err != nil {
		return errors.Wrap(err, "failed to get directory size")
	}
	log.Println("got size: ", size)
	if s.sizeLimit != 0 && size > int64(s.sizeLimit) {
		log.Printf("/!\\ Size limit is lower than current directory size (%dMiB > %dMiB). "+
			"Please remove extra files manually", size, s.sizeLimit)

		// TODO: uncomment - testing
		// s.sizeLimit = 0
	}
	go s.taskLimitSize()

	// Scheduled tasks
	scheduler := cron.New(
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DiscardLogger),
		))

	_, err = scheduler.AddFunc("@every 31m", s.taskPlaylistSync)
	if err != nil {
		return errors.Wrap(err, "failed to schedule task")
	}

	scheduler.Start()
	return nil
}
