package server

import (
	"fmt"
	"log"

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
	if s.sizeLimit != 0 {
		log.Printf("Max downloads size set to %d MiB\n", s.sizeLimit)
	}

	for {
		if _, ok := <-s.cleanup; ok {
			log.Println("taskLimitSize task triggered")

			if s.sizeLimit == 0 {
				continue
			}

			vids, err := getVideoFiles(s.downloadsDir, createdAsc)
			if err != nil {
				log.Println("Failed to get video files:", err)
				continue
			}

			over, diff := needsDeletion(vids, int64(s.sizeLimit))
			if !over {
				continue
			}

			var reclaimed int64
			vidsToDelete := make([]videoFile, 0, 2)
			for _, v := range vids {
				if reclaimed >= diff {
					break
				}

				reclaimed += v.Size
				vidsToDelete = append(vidsToDelete, v)
			}

			log.Printf("Over size limit by %d MiB. "+
				"Deleting %d videos to free %d MiB", diff, len(vidsToDelete), reclaimed)

			if err = deleteVideoFiles(s.fs, vidsToDelete, s.downloadsDir); err != nil {
				log.Println("Failed to delete videos:", err)
			}
		}
	}
}

func (s Server) triggerCleanup() {
	s.cleanup <- struct{}{}
}

func (s Server) startWorkers() error {
	// Startup tasks
	if s.playlistSyncURL != "" {
		log.Printf("Tracking playlist: %s\n", s.playlistSyncURL)
		s.taskPlaylistSync()
	}

	fmt.Println("test2")

	videos, err := getVideoFiles(s.downloadsDir, createdAsc)
	if err != nil {
		log.Fatalln("Failed to get video files:", err)
	}
	fmt.Println("test3")

	if over, diff := needsDeletion(videos, int64(s.sizeLimit)); over {
		log.Printf("/!\\ Size limit is lower than current directory size (by %d MiB). "+
			"Please remove extra files manually", diff)

		// TODO: uncomment - testing
		// s.sizeLimit = 0
	}
	go s.taskLimitSize()

	fmt.Println("test4")

	// Scheduled tasks
	scheduler := cron.New(
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DiscardLogger),
		))

	_, err = scheduler.AddFunc("@every 31m", s.taskPlaylistSync)
	if err != nil {
		return errors.Wrap(err, "failed to schedule task")
	}

	fmt.Println("test5")

	scheduler.Start()
	return nil
}
