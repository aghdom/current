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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"serve"},
	Short:   "Starts the application server",
	Long: `Current is a web application, which serves a site with all of the posts.
The 'server' command starts serving the application on the specified host and port.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(viper.Get("server.port"))
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	// Persistent Flags
	serveCmd.PersistentFlags().IntP("port", "p", 3773, "port on which the server will listen")

	// Binding Persistent Flags to Viper
	viper.BindPFlag("server.port", serveCmd.PersistentFlags().Lookup("port"))

	// Binding Environment Variables to Viper
	viper.BindEnv("server.port", "CRNT_SERVER_PORT")
}
