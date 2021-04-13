package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/esnet/grafana-dashboard-manager/config"
	"github.com/esnet/grafana-dashboard-manager/log"
	"github.com/grafana-tools/sdk"
	"github.com/thoas/go-funk"
)

//ListDashboards: List all dashboards optionally filtered by folder name. If folderFilters
// is blank, defaults to the configured Monitored folders
func ListDashboards(client *sdk.Client, folderFilters []string, query string) []sdk.FoundBoard {
	ctx := context.Background()
	var boardsList []sdk.FoundBoard = make([]sdk.FoundBoard, 0)
	boardLinks, err := client.SearchDashboards(ctx, query, false)
	if err != nil {
		panic(err)
	}
	if len(folderFilters) == 0 {
		folderFilters = config.GetGrafanaConfig().GetMonitoredFolders()
	}
	for _, link := range boardLinks {
		if funk.Contains(folderFilters, link.FolderTitle) {
			boardsList = append(boardsList, link)
		}

	}

	return boardsList

}

//ImportDashboards saves all dashboards matching query to configured location
func ImportDashboards(client *sdk.Client, query string, conf config.Provider) []string {
	var (
		boardLinks []sdk.FoundBoard
		rawBoard   []byte
		meta       sdk.BoardProperties
		err        error
	)
	ctx := context.Background()

	boardLinks = ListDashboards(client, config.GetGrafanaConfig().GetMonitoredFolders(), query)
	var boards []string = make([]string, 0)
	for _, link := range boardLinks {
		if rawBoard, meta, err = client.GetRawDashboardByUID(ctx, link.UID); err != nil {
			fmt.Fprintf(os.Stderr, "%s for %s\n", err, link.URI)
			continue
		}
		fileName := fmt.Sprintf("%s/%s.json", buildDashboardPath(conf, link.FolderTitle), meta.Slug)
		if err = ioutil.WriteFile(fileName, rawBoard, os.FileMode(int(0666))); err != nil {
			fmt.Fprintf(os.Stderr, "%s for %s\n", err, meta.Slug)
		} else {
			boards = append(boards, fileName)
		}

	}
	return boards
}

//getFolderNameIDMap helper function to build a mapping for name to folderID
func getFolderNameIDMap(client *sdk.Client, ctx context.Context) map[string]int {

	folders, _ := client.GetAllFolders(ctx)
	var folderMap map[string]int = make(map[string]int, 0)
	for _, folder := range folders {
		folderMap[folder.Title] = folder.ID
	}
	return folderMap
}

//ExportDashboards finds all the dashboards in the configured location and exports them to grafana.
// if the folde doesn't exist, it'll be created.
func ExportDashboards(client *sdk.Client, folderFilters []string, query string, conf config.Provider) {
	filesInDir := findAllFiles(getResourcePath(conf, "dashboard"))
	ctx := context.Background()
	var rawBoard []byte
	folderMap := getFolderNameIDMap(client, ctx)
	var err error
	var folderName string = ""
	var folderId int

	for _, file := range filesInDir {
		if strings.HasSuffix(file, ".json") {
			if rawBoard, err = ioutil.ReadFile(file); err != nil {
				log.Println(err)
				continue
			}
			elements := strings.Split(file, "/")
			if len(elements) >= 2 {
				folderName = elements[len(elements)-2]
			}
			if folderName == "" {
				folderId = sdk.DefaultFolderId
			} else {
				if val, ok := folderMap[folderName]; ok {
					folderId = val
				} else {
					folder := sdk.Folder{Title: folderName}
					folder, err = client.CreateFolder(ctx, folder)
					if err != nil {
						panic(err)
					}
					folderMap[folderName] = folder.ID
					folderId = folder.ID
				}
			}

			var board sdk.Board
			if err = json.Unmarshal(rawBoard, &board); err != nil {
				log.Println(err)
				log.Printf("Failed to unmarshall file: %s", file)
				continue
			}
			if _, err = client.DeleteDashboard(ctx, board.UpdateSlug()); err != nil {
				log.Println(err)
				continue
			}
			params := sdk.SetDashboardParams{
				FolderID:  folderId,
				Overwrite: false,
			}
			_, err := client.SetDashboard(ctx, board, params)
			if err != nil {
				log.Printf("error on Exporting dashboard %s", board.Title)
				continue
			}
		}
	}
}

//DeleteAllDashboards clears all current dashboards being monitored.  Any folder not white listed
// will not be affected
func DeleteAllDashboards(client *sdk.Client, folderFilters []string) []string {
	ctx := context.Background()
	var dashboards []string = make([]string, 0)
	items := ListDashboards(client, config.GetGrafanaConfig().GetMonitoredFolders(), "")
	for _, item := range items {
		_, err := client.DeleteDashboardByUID(ctx, item.UID)
		if err == nil {
			dashboards = append(dashboards, item.Title)
		}
	}
	return dashboards

}