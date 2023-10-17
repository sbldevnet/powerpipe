package cmd

import (
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/turbot/powerpipe/internal/service/api"
	"github.com/turbot/powerpipe/pkg/cmdconfig"
	"github.com/turbot/powerpipe/pkg/constants"
	"github.com/turbot/powerpipe/pkg/dashboard"
	"github.com/turbot/powerpipe/pkg/dashboard/dashboardserver"
	"github.com/turbot/powerpipe/pkg/error_helpers"
	"gopkg.in/olahol/melody.v1"
)

func serviceCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "service [command]",
		Args:  cobra.NoArgs,
		Short: "Powerpipe service management",
		Long: `Powerpipe service management.

Run Powerpipe as a local service, exposing it as a database endpoint for
connection from any compatible database client.`,
	}

	cmd.AddCommand(serviceStartCmd())
	cmd.Flags().BoolP(constants.ArgHelp, "h", false, "Help for service")
	return cmd
}

func serviceStartCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "start",
		Args:  cobra.NoArgs,
		Run:   runServiceStartCmd,
		Short: "Start Powerpipe in service mode",
		Long: `Start the Powerpipe service.

Run Powerpipe as a local service, exposing it as a database endpoint for
connection from any compatible database client.`,
	}

	cmdconfig.
		OnCmd(cmd).
		AddModLocationFlag().
		AddBoolFlag(constants.ArgHelp, false, "Help for service start", cmdconfig.FlagOptions.WithShortHand("h")).
		AddStringFlag(constants.ArgInstallDir, dashboard.DefaultInstallDir, "The default install directory")
	// AddStringFlag(constants.ArgConnectionString, "postgres://steampipe@localhost:9193/steampipe", "Database service port").
	// AddIntFlag(constants.ArgDatabasePort, constants.DatabaseDefaultPort, "Database service port").
	// AddStringFlag(constants.ArgDatabaseListenAddresses, string(db_local.ListenTypeNetwork), "Accept connections from: `local` (an alias for `localhost` only), `network` (an alias for `*`), or a comma separated list of hosts and/or IP addresses").
	// AddStringFlag(constants.ArgServicePassword, "", "Set the database password for this session").
	// // default is false and hides the database user password from service start prompt
	// AddBoolFlag(constants.ArgServiceShowPassword, false, "View database password for connecting from another machine").
	// // dashboard server
	// AddBoolFlag(constants.ArgDashboard, false, "Run the dashboard webserver with the service").
	// AddStringFlag(constants.ArgDashboardListen, string(dashboardserver.ListenTypeNetwork), "Accept connections from: local (localhost only) or network (open) (dashboard)").
	// AddIntFlag(constants.ArgDashboardPort, constants.DashboardServerDefaultPort, "Report server port").
	// // foreground enables the service to run in the foreground - till exit
	// AddBoolFlag(constants.ArgForeground, false, "Run the service in the foreground").

	// 	// flags relevant only if the --dashboard arg is used:
	// 	AddStringSliceFlag(constants.ArgVarFile, nil, "Specify an .spvar file containing variable values (only applies if '--dashboard' flag is also set)").
	// 	// NOTE: use StringArrayFlag for ArgVariable, not StringSliceFlag
	// 	// Cobra will interpret values passed to a StringSliceFlag as CSV,
	// 	// where args passed to StringArrayFlag are not parsed and used raw
	// 	AddStringArrayFlag(constants.ArgVariable, nil, "Specify the value of a variable (only applies if '--dashboard' flag is also set)").

	// 	// hidden flags for internal use
	// 	AddStringFlag(constants.ArgInvoker, string(constants.InvokerService), "Invoked by \"service\" or \"query\"", cmdconfig.FlagOptions.Hidden())

	return cmd
}

func runServiceStartCmd(cmd *cobra.Command, _ []string) {
	dashboard.PowerpipeDir = "~/.powerpipe"

	ctx := cmd.Context()
	ctx, stopFn := signal.NotifyContext(ctx, os.Interrupt)
	defer stopFn()

	modInitData := dashboard.InitDashboard(ctx)
	if modInitData.Result.Error != nil {
		panic(modInitData.Result.Error)
	}

	err := dashboard.Ensure(ctx)
	if err != nil {
		panic(err)
	}

	// setup a new webSocket service
	webSocket := melody.New()

	// create the dashboardServer
	dashboardServer, err := dashboardserver.NewServer(ctx, modInitData.Client, modInitData.Workspace, webSocket)
	error_helpers.FailOnError(err)

	// send it over to the API Server
	apiServer, err := api.NewAPIService(ctx, api.WithWebSocket(webSocket))
	if err != nil {
		panic(err)
	}
	dashboardServer.InitAsync(ctx)

	// start the API server
	go apiServer.Start()
	println("server started")
	<-ctx.Done()

	// dashboard.InitDashboard(ctx)
}
