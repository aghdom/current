/*
Copyright © 2022 Dominik Ágh <agh.dominik@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/aghdom/current/server"
)

// serverCmd represents the serve command
var serverCmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"serve"},
	Short:   "Starts the application server",
	Long: `Current is a web application, which serves a site with all of the posts.
The 'server' command starts serving the application on the specified host and port.`,
	Run: runServer,
}

func runServer(cmd *cobra.Command, args []string) {
	server.Run()
}

func init() {
	rootCmd.AddCommand(serverCmd)
	// Persistent Flags
	serverCmd.PersistentFlags().String("host", "localhost", "host address on which the server will listen")
	serverCmd.PersistentFlags().IntP("port", "p", 3773, "port on which the server will listen")

	// Binding Persistent Flags to Viper
	viper.BindPFlag("server.port", serverCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("server.host", serverCmd.PersistentFlags().Lookup("host"))

	// Binding Environment Variables to Viper
	viper.BindEnv("server.port", "CRNT_SERVER_PORT")
	viper.BindEnv("server.host", "CRNT_SERVER_HOST")

}
