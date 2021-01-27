package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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

func (s Server) taskLimitSize() {
	log.Println("taskLimitSize task triggered")

	go func() {

		time.AfterFunc(2*time.Second, func() {
			fmt.Println("sending")
			s.downloadFinish <- struct{}{}
			fmt.Println("sent")

		})
	}()

	for {
		fmt.Println("listening")
		j, more := <-s.downloadFinish
		if more {
			fmt.Println("received job", j)

			if s.sizeLimit == 0 {
				continue
			}

		} else {
			fmt.Println("received all jobs")
			return
		}
	}

}

// DirSize returns a directory's size in mB
const toMB = 1024 * 1024

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
	return size / toMB, err
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
	if s.sizeLimit != 0 {
		if size > int64(s.sizeLimit) {
			log.Printf("/!\\ Size limit is lower than current directory size (%dmB > %dmB). "+
				"Please remove extra files manually", size, s.sizeLimit)
			s.sizeLimit = 0
		}

		go s.taskLimitSize()
	}

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
