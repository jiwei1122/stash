package manager

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/stashapp/stash/pkg/database"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/manager/config"
	"github.com/stashapp/stash/pkg/models"
)

type CleanTask struct {
	Scene   *models.Scene
	Gallery *models.Gallery
}

func (t *CleanTask) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	if t.Scene != nil && t.shouldClean(t.Scene.Path) {
		t.deleteScene(t.Scene.ID)
	}

	if t.Gallery != nil && t.shouldClean(t.Gallery.Path) {
		t.deleteGallery(t.Gallery.ID)
	}
}

func (t *CleanTask) shouldClean(path string) bool {
	if t.fileExists(path) && t.pathInStash(path) {
		logger.Debugf("File Found: %s", path)
		if matchFile(path, config.GetExcludes()) {
			logger.Infof("File matched regex. Cleaning: \"%s\"", path)
			return true
		}
	} else {
		logger.Infof("File not found. Cleaning: \"%s\"", path)
		return true
	}

	return false
}

func (t *CleanTask) deleteScene(sceneID int) {
	ctx := context.TODO()
	qb := models.NewSceneQueryBuilder()
	tx := database.DB.MustBeginTx(ctx, nil)

	scene, err := qb.Find(sceneID)
	err = DestroyScene(sceneID, tx)

	if err != nil {
		logger.Infof("Error deleting scene from database: %s", err.Error())
		tx.Rollback()
		return
	}

	if err := tx.Commit(); err != nil {
		logger.Infof("Error deleting scene from database: %s", err.Error())
		return
	}

	DeleteGeneratedSceneFiles(scene)
}

func (t *CleanTask) deleteGallery(galleryID int) {
	ctx := context.TODO()
	qb := models.NewGalleryQueryBuilder()
	tx := database.DB.MustBeginTx(ctx, nil)

	err := qb.Destroy(galleryID, tx)

	if err != nil {
		logger.Infof("Error deleting gallery from database: %s", err.Error())
		tx.Rollback()
		return
	}

	if err := tx.Commit(); err != nil {
		logger.Infof("Error deleting gallery from database: %s", err.Error())
		return
	}
}

func (t *CleanTask) fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (t *CleanTask) pathInStash(pathToCheck string) bool {
	for _, path := range config.GetStashPaths() {

		rel, error := filepath.Rel(path, filepath.Dir(pathToCheck))

		if error == nil {
			if !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				logger.Debugf("File %s belongs to stash path %s", pathToCheck, path)
				return true
			}
		}

	}
	logger.Debugf("File %s is out from stash path", pathToCheck)
	return false
}
