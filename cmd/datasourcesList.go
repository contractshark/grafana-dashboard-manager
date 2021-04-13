package cmd

import (
	"fmt"

	"github.com/esnet/grafana-dashboard-manager/api"
	"github.com/esnet/grafana-dashboard-manager/config"
	"github.com/jedib0t/go-pretty/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var listDataSources = &cobra.Command{
	Use:   "list",
	Short: "List all dashboards",
	Long:  `List all dashboards`,
	Run: func(cmd *cobra.Command, args []string) {
		tableObj.AppendHeader(table.Row{"id", "name", "type", "default", "url"})

		datasources := api.ListDataSources(client, nil)
		if len(datasources) == 0 {
			log.Info("No datasources found")
		} else {
			for _, link := range datasources {
				url := fmt.Sprintf("%s/datasource/edit/%d", config.GetGrafanaConfig().URL, link.ID)
				tableObj.AppendRow(table.Row{link.ID, link.Name, link.Type, link.IsDefault, url})
			}
			tableObj.Render()
		}
	},
}

func init() {
	datasources.AddCommand(listDataSources)
}