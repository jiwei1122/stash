// +build integration

package models_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/stashapp/stash/pkg/database"
	"github.com/stashapp/stash/pkg/models"
)

const totalScenes = 12
const totalPerformers = 3
const totalMovies = 1
const totalGalleries = 1
const totalTags = 5
const totalStudios = 1

var sceneIDs []int
var performerIDs []int
var movieIDs []int
var galleryIDs []int
var tagIDs []int
var studioIDs []int
var markerIDs []int

const sceneIdxWithMovie = 0
const sceneIdxWithGallery = 1
const sceneIdxWithPerformer = 2
const sceneIdxWithTwoPerformers = 3
const sceneIdxWithTag = 4
const sceneIdxWithTwoTags = 5
const sceneIdxWithStudio = 6
const sceneIdxWithMarker = 7

const performerIdxWithScene = 0

const movieIdxWithScene = 0

const galleryIdxWithScene = 0

const performerIdx1WithScene = 1
const performerIdx2WithScene = 2

const tagIdxWithScene = 0
const tagIdx1WithScene = 1
const tagIdx2WithScene = 2
const tagIdxWithPrimaryMarker = 3
const tagIdxWithMarker = 4

const studioIdxWithScene = 0

const markerIdxWithScene = 0

const pathField = "Path"
const checksumField = "Checksum"
const titleField = "Title"

func TestMain(m *testing.M) {
	ret := runTests(m)
	os.Exit(ret)
}

func testTeardown(databaseFile string) {
	err := database.DB.Close()

	if err != nil {
		panic(err)
	}

	err = os.Remove(databaseFile)
	if err != nil {
		panic(err)
	}
}

func runTests(m *testing.M) int {
	// create the database file
	f, err := ioutil.TempFile("", "*.sqlite")
	if err != nil {
		panic(fmt.Sprintf("Could not create temporary file: %s", err.Error()))
	}

	f.Close()
	databaseFile := f.Name()
	database.Initialize(databaseFile)

	// defer close and delete the database
	defer testTeardown(databaseFile)

	err = populateDB()
	if err != nil {
		panic(fmt.Sprintf("Could not populate database: %s", err.Error()))
	} else {
		// run the tests
		return m.Run()
	}
}

func populateDB() error {
	ctx := context.TODO()
	tx := database.DB.MustBeginTx(ctx, nil)

	if err := createScenes(tx, totalScenes); err != nil {
		tx.Rollback()
		return err
	}

	if err := createGalleries(tx, totalGalleries); err != nil {
		tx.Rollback()
		return err
	}

	if err := createMovies(tx, totalMovies); err != nil {
		tx.Rollback()
		return err
	}

	if err := createPerformers(tx, totalPerformers); err != nil {
		tx.Rollback()
		return err
	}

	if err := createTags(tx, totalTags); err != nil {
		tx.Rollback()
		return err
	}

	if err := createStudios(tx, totalStudios); err != nil {
		tx.Rollback()
		return err
	}

	// TODO - the link methods use Find which don't accept a transaction, so
	// to commit the transaction and start a new one
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("Error committing: %s", err.Error())
	}

	tx = database.DB.MustBeginTx(ctx, nil)

	if err := linkSceneGallery(tx, sceneIdxWithGallery, galleryIdxWithScene); err != nil {
		tx.Rollback()
		return err
	}

	if err := linkSceneMovie(tx, sceneIdxWithMovie, movieIdxWithScene); err != nil {
		tx.Rollback()
		return err
	}

	if err := linkScenePerformers(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := linkSceneTags(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := linkSceneStudio(tx, sceneIdxWithStudio, studioIdxWithScene); err != nil {
		tx.Rollback()
		return err
	}

	if err := createMarker(tx, sceneIdxWithMarker, tagIdxWithPrimaryMarker, []int{tagIdxWithMarker}); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("Error committing: %s", err.Error())
	}

	return nil
}

func getSceneStringValue(index int, field string) string {
	return fmt.Sprintf("scene_%04d_%s", index, field)
}

func getSceneRating(index int) sql.NullInt64 {
	rating := index % 6
	return sql.NullInt64{Int64: int64(rating), Valid: rating > 0}
}

func getSceneOCounter(index int) int {
	return index % 3
}

func getSceneDuration(index int) sql.NullFloat64 {
	duration := index % 4
	duration = duration * 100

	return sql.NullFloat64{
		Float64: float64(duration) + 0.432,
		Valid:   duration != 0,
	}
}

func getSceneHeight(index int) sql.NullInt64 {
	heights := []int64{0, 200, 240, 300, 480, 700, 720, 800, 1080, 1500, 2160, 3000}
	height := heights[index%len(heights)]
	return sql.NullInt64{
		Int64: height,
		Valid: height != 0,
	}
}

func getSceneDate(index int) models.SQLiteDate {
	dates := []string{"null", "", "0001-01-01", "2001-02-03"}
	date := dates[index%len(dates)]
	return models.SQLiteDate{
		String: date,
		Valid:  date != "null",
	}
}

func createScenes(tx *sqlx.Tx, n int) error {
	sqb := models.NewSceneQueryBuilder()

	for i := 0; i < n; i++ {
		scene := models.Scene{
			Path:     getSceneStringValue(i, pathField),
			Title:    sql.NullString{String: getSceneStringValue(i, titleField), Valid: true},
			Checksum: getSceneStringValue(i, checksumField),
			Details:  sql.NullString{String: getSceneStringValue(i, "Details"), Valid: true},
			Rating:   getSceneRating(i),
			OCounter: getSceneOCounter(i),
			Duration: getSceneDuration(i),
			Height:   getSceneHeight(i),
			Date:     getSceneDate(i),
		}

		created, err := sqb.Create(scene, tx)

		if err != nil {
			return fmt.Errorf("Error creating scene %v+: %s", scene, err.Error())
		}

		sceneIDs = append(sceneIDs, created.ID)
	}

	return nil
}

func getGalleryStringValue(index int, field string) string {
	return "gallery_" + strconv.FormatInt(int64(index), 10) + "_" + field
}

func createGalleries(tx *sqlx.Tx, n int) error {
	gqb := models.NewGalleryQueryBuilder()

	for i := 0; i < n; i++ {
		gallery := models.Gallery{
			Path:     getGalleryStringValue(i, pathField),
			Checksum: getGalleryStringValue(i, checksumField),
		}

		created, err := gqb.Create(gallery, tx)

		if err != nil {
			return fmt.Errorf("Error creating gallery %v+: %s", gallery, err.Error())
		}

		galleryIDs = append(galleryIDs, created.ID)
	}

	return nil
}

func getMovieStringValue(index int, field string) string {
	return "movie_" + strconv.FormatInt(int64(index), 10) + "_" + field
}

func createMovies(tx *sqlx.Tx, n int) error {
	mqb := models.NewMovieQueryBuilder()

	for i := 0; i < n; i++ {
		movie := models.Movie{
			Name:       sql.NullString{String: getMovieStringValue(i, "Name"), Valid: true},
			FrontImage: []byte(models.DefaultMovieImage),
		}

		created, err := mqb.Create(movie, tx)

		if err != nil {
			return fmt.Errorf("Error creating movie %v+: %s", movie, err.Error())
		}

		movieIDs = append(movieIDs, created.ID)
	}

	return nil
}

func getPerformerStringValue(index int, field string) string {
	return "performer_" + strconv.FormatInt(int64(index), 10) + "_" + field
}

func getPerformerBoolValue(index int) bool {
	index = index % 2
	return index == 1
}

func createPerformers(tx *sqlx.Tx, n int) error {
	pqb := models.NewPerformerQueryBuilder()

	for i := 0; i < n; i++ {
		performer := models.Performer{
			Name:     sql.NullString{String: getPerformerStringValue(i, "Name"), Valid: true},
			Checksum: getPerformerStringValue(i, checksumField),
			// just use movie image
			Image:    []byte(models.DefaultMovieImage),
			Favorite: sql.NullBool{Bool: getPerformerBoolValue(i), Valid: true},
		}

		created, err := pqb.Create(performer, tx)

		if err != nil {
			return fmt.Errorf("Error creating performer %v+: %s", performer, err.Error())
		}

		performerIDs = append(performerIDs, created.ID)
	}

	return nil
}

func getTagStringValue(index int, field string) string {
	return "tag_" + strconv.FormatInt(int64(index), 10) + "_" + field
}

func createTags(tx *sqlx.Tx, n int) error {
	tqb := models.NewTagQueryBuilder()

	for i := 0; i < n; i++ {
		tag := models.Tag{
			Name: getTagStringValue(i, "Name"),
		}

		created, err := tqb.Create(tag, tx)

		if err != nil {
			return fmt.Errorf("Error creating tag %v+: %s", tag, err.Error())
		}

		tagIDs = append(tagIDs, created.ID)
	}

	return nil
}

func getStudioStringValue(index int, field string) string {
	return "studio_" + strconv.FormatInt(int64(index), 10) + "_" + field
}

func createStudios(tx *sqlx.Tx, n int) error {
	sqb := models.NewStudioQueryBuilder()

	for i := 0; i < n; i++ {
		tag := models.Studio{
			Name:  sql.NullString{String: getStudioStringValue(i, "Name"), Valid: true},
			Image: []byte(models.DefaultStudioImage),
		}

		created, err := sqb.Create(tag, tx)

		if err != nil {
			return fmt.Errorf("Error creating studio %v+: %s", tag, err.Error())
		}

		studioIDs = append(studioIDs, created.ID)
	}

	return nil
}

func createMarker(tx *sqlx.Tx, sceneIdx, primaryTagIdx int, tagIdxs []int) error {
	mqb := models.NewSceneMarkerQueryBuilder()

	marker := models.SceneMarker{
		SceneID:      sql.NullInt64{Int64: int64(sceneIDs[sceneIdx]), Valid: true},
		PrimaryTagID: tagIDs[primaryTagIdx],
	}

	created, err := mqb.Create(marker, tx)

	if err != nil {
		return fmt.Errorf("Error creating marker %v+: %s", marker, err.Error())
	}

	markerIDs = append(markerIDs, created.ID)

	jqb := models.NewJoinsQueryBuilder()

	joins := []models.SceneMarkersTags{}

	for _, tagIdx := range tagIdxs {
		join := models.SceneMarkersTags{
			SceneMarkerID: created.ID,
			TagID:         tagIDs[tagIdx],
		}
		joins = append(joins, join)
	}

	if err := jqb.CreateSceneMarkersTags(joins, tx); err != nil {
		return fmt.Errorf("Error creating marker/tag join: %s", err.Error())
	}

	return nil
}

func linkSceneMovie(tx *sqlx.Tx, sceneIndex, movieIndex int) error {
	jqb := models.NewJoinsQueryBuilder()

	_, err := jqb.AddMoviesScene(sceneIDs[sceneIndex], movieIDs[movieIndex], nil, tx)
	return err
}

func linkScenePerformers(tx *sqlx.Tx) error {
	if err := linkScenePerformer(tx, sceneIdxWithPerformer, performerIdxWithScene); err != nil {
		return err
	}
	if err := linkScenePerformer(tx, sceneIdxWithTwoPerformers, performerIdx1WithScene); err != nil {
		return err
	}
	if err := linkScenePerformer(tx, sceneIdxWithTwoPerformers, performerIdx2WithScene); err != nil {
		return err
	}

	return nil
}

func linkScenePerformer(tx *sqlx.Tx, sceneIndex, performerIndex int) error {
	jqb := models.NewJoinsQueryBuilder()

	_, err := jqb.AddPerformerScene(sceneIDs[sceneIndex], performerIDs[performerIndex], tx)
	return err
}

func linkSceneGallery(tx *sqlx.Tx, sceneIndex, galleryIndex int) error {
	gqb := models.NewGalleryQueryBuilder()

	gallery, err := gqb.Find(galleryIDs[galleryIndex])

	if err != nil {
		return fmt.Errorf("error finding gallery: %s", err.Error())
	}

	if gallery == nil {
		return errors.New("gallery is nil")
	}

	gallery.SceneID = sql.NullInt64{Int64: int64(sceneIDs[sceneIndex]), Valid: true}
	_, err = gqb.Update(*gallery, tx)

	return err
}

func linkSceneTags(tx *sqlx.Tx) error {
	if err := linkSceneTag(tx, sceneIdxWithTag, tagIdxWithScene); err != nil {
		return err
	}
	if err := linkSceneTag(tx, sceneIdxWithTwoTags, tagIdx1WithScene); err != nil {
		return err
	}
	if err := linkSceneTag(tx, sceneIdxWithTwoTags, tagIdx2WithScene); err != nil {
		return err
	}

	return nil
}

func linkSceneTag(tx *sqlx.Tx, sceneIndex, tagIndex int) error {
	jqb := models.NewJoinsQueryBuilder()

	_, err := jqb.AddSceneTag(sceneIDs[sceneIndex], tagIDs[tagIndex], tx)
	return err
}

func linkSceneStudio(tx *sqlx.Tx, sceneIndex, studioIndex int) error {
	sqb := models.NewSceneQueryBuilder()

	scene := models.ScenePartial{
		ID:       sceneIDs[sceneIndex],
		StudioID: &sql.NullInt64{Int64: int64(studioIDs[studioIndex]), Valid: true},
	}
	_, err := sqb.Update(scene, tx)

	return err
}
