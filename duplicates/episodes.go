package duplates

import (
	"log"
	"sort"
	"strings"

	"github.com/jinzhu/gorm"

	"github.com/uhuraapp/uhura-api/models"
)

type episode struct {
	Title string
	Count int64
	ID    int64
}

// Episodes find duplicated episodes
func Episodes(DB gorm.DB, del chan int64, cl chan bool) {
	var episodes []episode

	DB.Table(models.Episode{}.TableName()).Select("items.title as title, o.dupeCount as count, items.id as id").Joins("INNER JOIN (SELECT title, channel_id, COUNT(*) as dupeCount FROM items GROUP BY title,channel_id HAVING COUNT(*) > 1) o on o.title = items.title AND o.channel_id = items.channel_id").Scan(&episodes)

	log.Println("SQL FOUND DUP", episodes)
	organizedEpisodes := organizeDuplicates(episodes)
	log.Println("ORGIZED", organizedEpisodes)

	for _, es := range organizedEpisodes {
		e, others := lastAndOthersEpisodes(es)
		log.Println("--- FIRST: ", e)
		log.Println("--- OTHERS: ", others)
		updatePlays(e, others, DB)
		for _, other := range others {
			del <- other.ID
		}
	}

	cl <- true
}

//
func organizeDuplicates(episodes []episode) map[string][]episode {
	duplicateEpisodes := make(map[string][]episode)
	for _, e := range episodes {
		key := strings.ToLower(e.Title)

		if len(duplicateEpisodes[key]) == 0 {
			duplicateEpisodes[key] = make([]episode, 0)
		}
		duplicateEpisodes[key] = append(duplicateEpisodes[key], e)
	}
	return duplicateEpisodes
}

func lastAndOthersEpisodes(episodes []episode) (episode, []episode) {
	sort.Sort(episodeByID(episodes))
	var newEpisodes []episode
	if len(episodes) > 1 {
		newEpisodes = episodes[:len(episodes)-1]
	}
	// check source
	return episodes[len(episodes)-1], newEpisodes
}

func updatePlays(e episode, others []episode, DB gorm.DB) {
	var ids []int64
	var playsIDs []int64

	for _, o := range others {
		ids = append(ids, o.ID)
	}

	plays := getPlays(ids, DB)

	for _, p := range plays {
		playsIDs = append(playsIDs, p.Id)
	}

	log.Println(" ---------- plays", playsIDs)
	if len(playsIDs) > 0 {
		DB.Table(models.Listened{}.TableName()).Where("id in (?)", playsIDs).Update("item_id", e.ID)
	}
}

func getPlays(ids []int64, DB gorm.DB) []models.Listened {
	plays := make([]models.Listened, 0)
	if len(ids) > 0 {
		DB.Table(models.Listened{}.TableName()).Where("item_id in (?)", ids).Find(&plays)
	}
	return plays
}

//

type episodeByID []episode

func (a episodeByID) Len() int           { return len(a) }
func (a episodeByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a episodeByID) Less(i, j int) bool { return a[i].ID < a[j].ID }
