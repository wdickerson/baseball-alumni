package main

import (
    "time"
    "html/template"
    "encoding/json"
    "fmt"
    "sort"
    "os"
    "log"
    "io/ioutil"
    "net/http"
)

type Team struct {
    Name string
    ID json.Number
}

type Player struct {
    FullName string
    CurrentTeam struct {
        ID json.Number
    }
}

type MlbPeopleResponse struct {
    People []Player
}

type Game struct {
    GamePk int
    GameDate string
    Teams struct {
        Away struct {
            IsWinner bool
            Score json.Number
            Team Team
        }
        Home struct {
            IsWinner bool
            Score json.Number
            Team Team
        }
    }
    Status struct {
        StatusCode string
    }
}

type MlbScheduleResponse struct {
    Copyright string
    Dates []Day
}

type PlayerData struct {
    PlayerID string
    UrlName string
    Updated time.Time
    MlbData Player
}

type MlbTeamResponse struct {
    Teams []Team
}

type Day struct {
    Date string
    Games []Game
}

type TeamData struct {
    ID json.Number
    Name string
    Bulldogs []string
    Schedule []Day
    LastGames []Game
    NextGames []Game
}

func (p *PlayerData) addPlayerToTeam() {
    // lets update teamStore with the bulldog
    currentTeamID := string(p.MlbData.CurrentTeam.ID)
    teamStore[currentTeamID].Bulldogs = append(teamStore[currentTeamID].Bulldogs, p.PlayerID)
    teamStore[currentTeamID].updateSchedule();
}

func (p *PlayerData) updatePlayerMlbData() {
    
    fmt.Println("lets update", p.UrlName);
    
    // app-people-1.json
    response, err := http.Get(fmt.Sprintf("https://statsapi.mlb.com/api/v1/people/%s?hydrate=currentTeam,team,stats(type=[yearByYear,careerRegularSeason,availableStats](team(league)),leagueListId=mlb_hist)&site=en", p.PlayerID))

    if err != nil {
        fmt.Printf("The HTTP request failed with error %s\n", err)
        panic(err)
    } else {
        body, _ := ioutil.ReadAll(response.Body)
        mlbPeopleResponse := MlbPeopleResponse{}
        json.Unmarshal(body, &mlbPeopleResponse)
        p.Updated = time.Now()
        p.MlbData = mlbPeopleResponse.People[0]
    }
}

func (myTeam *TeamData) updateSchedule() {
    
    fmt.Println("update sched for", myTeam.Name)
    
    // Lets get the schedule/results for this players team
    // app-schedule-1.json
    response, err := http.Get(fmt.Sprintf("https://statsapi.mlb.com/api/v1/schedule/?sportId=1&teamId=%s&season=2018&startDate=2018-01-01&endDate=2018-12-31", myTeam.ID))

    if err != nil {
        fmt.Printf("The HTTP request failed with error %s\n", err)
        panic(err)
    } else {
        body, _ := ioutil.ReadAll(response.Body)
        scheduleResponse := MlbScheduleResponse{}
        json.Unmarshal(body, &scheduleResponse)

        myTeam.Schedule = scheduleResponse.Dates

        // Find where today falls in the array of days
        currentDayIndex := 0;
        for _, day := range myTeam.Schedule {
            if day.Date < CURRENT_TIME.Format("2006-01-02") {
                currentDayIndex++;
            }
        }

        // say currentDayInex is 7
        // we need to poulate LastGames with
        // Day[7] (conditionally)
        // Day[6]
        // Day[5]
        // ...
        // untill we reach three games
        //
        // we need to poulate NextGames with
        // Day[7] (conditionally)
        // Day[8]
        // Day[9]
        // ...
        // untill we reach three games

        // lets populate last 3
        for j := currentDayIndex; j >= 0; j-- {
            if len(myTeam.LastGames) >= 3 {
                break;
            }

            for _, game := range myTeam.Schedule[j].Games {
                // if the game status is finished, add it to LastGames
                if len(myTeam.LastGames) >= 3 {
                    break;
                }

                // Would rather look at StatusCode, but using a mock time
                // in the past for CURRENT_TIME so we have to compare dates
                gameTime, _ := time.Parse(time.RFC3339, game.GameDate);
                if (gameTime.Before(CURRENT_TIME)) {
                    myTeam.LastGames = append(myTeam.LastGames, game);
                }

//                if (game.Status.StatusCode == "F") {
//                    myTeam.LastGames = append(myTeam.LastGames, game);
//                }
            }
        }

        sort.Slice(myTeam.LastGames, func(i, j int) bool {
            return myTeam.LastGames[i].GameDate < myTeam.LastGames[j].GameDate
        })

        // lets populate next 3
        for j := currentDayIndex; j < len(myTeam.Schedule); j++ {
            if len(myTeam.NextGames) >= 3 {
                break;
            }

            for _, game := range myTeam.Schedule[j].Games {
                // if the game status is not finished, add it to LastGames
                if len(myTeam.NextGames) >= 3 {
                    break;
                }

                // Would rather look at StatusCode, but using a mock time
                // in the past for CURRENT_TIME so we have to compare dates
                gameTime, _ := time.Parse(time.RFC3339, game.GameDate);
                if (gameTime.After(CURRENT_TIME)) {
                    myTeam.NextGames = append(myTeam.NextGames, game);
                }

//                if (game.Status.StatusCode != "F") {
//                    myTeam.NextGames = append(myTeam.NextGames, game);
//                }
            }
        }

        sort.Slice(myTeam.NextGames, func(i, j int) bool {
            return myTeam.NextGames[i].GameDate < myTeam.NextGames[j].GameDate
        })
    }
}

func initializeTeamStore() {
    // app-teams-1.json
    response, err := http.Get("https://statsapi.mlb.com/api/v1/teams?sportId=1&language=en&leagueListId=mlb_hist&activeStatus=B&season=2019")

    if err != nil {
        fmt.Printf("The HTTP request failed with error %s\n", err)
        panic(err)
    } else {
        body, _ := ioutil.ReadAll(response.Body)
        mlbTeamResponse := MlbTeamResponse{}
        json.Unmarshal(body, &mlbTeamResponse)

        for k := range mlbTeamResponse.Teams {
            teamStore[string(mlbTeamResponse.Teams[k].ID)] = &TeamData{};
            teamStore[string(mlbTeamResponse.Teams[k].ID)].Name = mlbTeamResponse.Teams[k].Name;
            teamStore[string(mlbTeamResponse.Teams[k].ID)].ID = mlbTeamResponse.Teams[k].ID;
        }
    }
}

func updateGameStore() {    
    gameStore.LastDays = []Day{}
    gameStore.NextDays = []Day{}
    
    // initialize the gameStore with the days of interest
    for i := 0; i <= 3; i++ {
        var newDay = Day{CURRENT_TIME.AddDate(0, 0, i).Format("2006-01-02"), []Game{}}
        gameStore.NextDays = append(gameStore.NextDays, newDay)
    }
    
    for i := 3; i > 0; i-- {
        var newDay = Day{CURRENT_TIME.AddDate(0, 0, -i).Format("2006-01-02"), []Game{}}
        gameStore.LastDays = append(gameStore.LastDays, newDay)
    }
    
    for teamID, teamData := range teamStore {
        if len(teamData.Bulldogs) > 0 {
            // we need to make sure this teams schedule is up-to-date
            teamStore[teamID].updateSchedule();
            
            for i, myDay := range gameStore.LastDays {
                for _, teamDay := range teamData.Schedule {
                    if myDay.Date == teamDay.Date {
                        gameStore.LastDays[i].Games = AppendUniqueGame(myDay.Games, teamDay.Games...)
                    }
                }
            }
            
            for i, myDay := range gameStore.NextDays {
                for _, teamDay := range teamData.Schedule {
                    if myDay.Date == teamDay.Date {
                        gameStore.NextDays[i].Games = AppendUniqueGame(myDay.Games, teamDay.Games...)
                    }
                }
            }
        }
    }
    
    // lets sort LastDays games by start time
    for k := range gameStore.LastDays {
        sort.Slice(gameStore.LastDays[k].Games, func(i, j int) bool {
            return gameStore.LastDays[k].Games[i].GameDate < gameStore.LastDays[k].Games[j].GameDate
        }) 
    }
    
    // lets sort NextDays games by start time
    for k := range gameStore.NextDays {
        sort.Slice(gameStore.NextDays[k].Games, func(i, j int) bool {
            return gameStore.NextDays[k].Games[i].GameDate < gameStore.NextDays[k].Games[j].GameDate
        }) 
    }
}

func AppendUniqueGame(gameList []Game, games ...Game) []Game {
    for _, newGame := range games {
        exists := false
        for _, existingGame := range gameList {
            if existingGame.GamePk == newGame.GamePk {
                exists = true;
            }
        }
        
        if !exists {
            gameList = append(gameList, newGame);
        }
    }
    
    return gameList;
}

func cacher(f http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        playerId := PLAYER_IDS[r.URL.Path[len("/player/"):]]
        p := playerStore[playerId]
        fmt.Println("Cache age: " + fmt.Sprintf("%.6f", time.Since(p.Updated).Seconds()))

        if time.Since(p.Updated).Seconds() > 30 {
            p.updatePlayerMlbData();
        }

        // call the next handler
        f(w, r)
    }
}

func handler(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("home.html")
    t.Execute(w, map[string]interface{}{
        "PlayerStore": playerStore,
        "GameStore": gameStore,
    });
}

func playersHandler(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("players.html")
    t.Execute(w, map[string]interface{}{
        "PlayerStore": playerStore,
    });
}

func playerHandler(w http.ResponseWriter, r *http.Request) {
    playerId := PLAYER_IDS[r.URL.Path[len("/player/"):]]
    myPlayer := playerStore[playerId];
    t, _ := template.ParseFiles("player.html")
    t.Execute(w, map[string]interface{}{
        "PlayerData": myPlayer,
        "TeamData": teamStore[string(myPlayer.MlbData.CurrentTeam.ID)],
    })
}

var playerStore = make(map[string]*PlayerData)
var teamStore = make(map[string]*TeamData)
var CURRENT_TIME time.Time
var PLAYER_IDS map[string]string
var gameStore struct {
    Updated time.Time
    LastDays []Day
    NextDays []Day
}

func main() {
    fmt.Println("This is project cupcake!")
    CURRENT_TIME, _ = time.Parse(time.RFC3339, "2018-03-01T09:00:00Z")

    // initialize the teamStore
    initializeTeamStore()

    // lets load players from json
    jsonFile, _ := os.Open("players.json")
    defer jsonFile.Close()
    byteValue, _ := ioutil.ReadAll(jsonFile)
    json.Unmarshal([]byte(byteValue), &PLAYER_IDS)

    for k := range PLAYER_IDS {
        playerStore[PLAYER_IDS[k]] = &PlayerData{}
        playerStore[PLAYER_IDS[k]].PlayerID = PLAYER_IDS[k]
        playerStore[PLAYER_IDS[k]].UrlName = k
        playerStore[PLAYER_IDS[k]].updatePlayerMlbData()
        playerStore[PLAYER_IDS[k]].addPlayerToTeam()
    }
    
    // lets get the game in the past few days and next
    // few days that feature bulldogs
    updateGameStore()

    staticFs := http.FileServer(http.Dir("assets"))
    http.Handle("/assets/", http.StripPrefix("/assets/", staticFs))

    http.HandleFunc("/", handler)
    http.HandleFunc("/player/", cacher(playerHandler))
    http.HandleFunc("/players/", playersHandler)

    log.Fatal(http.ListenAndServe(":8080", nil))
    fmt.Println("Terminating the application...")
}
