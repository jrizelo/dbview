// Copyright © 2017 uMov.me Team <devteam-umovme@googlegroups.com>
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors
//    may be used to endorse or promote products derived from this software
//    without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/umovme/dbview/setup"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the dbview in the database",
	Long: `

	Install all dependencies of the dbview environment like, 
users, permissions, database and restores the database dump.
	
The database dump are provided by the uMov.me support team. 
	
Please contact us with you have any trouble.`,
	Run: func(cmd *cobra.Command, args []string) {

		log("Validating parameters...")
		if !checkInputParameters() {
			return
		}
		conn := setup.ConnectionDetails{Username: pDBUserName, Host: pDBHost, Database: pDBName, SslMode: pDBSslMode}

		customerUser := fmt.Sprintf("u%d", pCustomerID)
		cleanup(conn, customerUser)

		for _, user := range []string{pTargetUserName, customerUser} {
			log("Creating the '%s' user", user)
			abort(
				setup.CreateUser(conn, user, nil))
		}

		log("Fixing permissions")
		abort(
			setup.GrantRolesToUser(conn, customerUser, []string{pTargetUserName}))

		log("Updating the 'search_path'")
		abort(
			setup.SetSearchPathForUser(conn, customerUser, []string{customerUser, "public"}))

		log("Creating the '%s' database", pTargetDatabase)
		abort(
			setup.CreateNewDatabase(conn, pTargetDatabase, []string{"OWNER " + pTargetUserName, "TEMPLATE template0"}))

		log("Creating the necessary extensions")
		conn.Database = pTargetDatabase
		abort(
			setup.CreateExtensionsInDatabase(conn, []string{"hstore", "dblink", "pg_freespacemap", "postgis", "tablefunc", "unaccent"}))

		exists, err := setup.CheckIfSchemaExists(conn, "dbview")
		abort(err)

		restoreArgs := []string{"-Fc"}

		if exists {
			// if exists the dbview schema, this is not a first user schema on this database
			// then just create a new schema and restore only it
			abort(
				setup.CreateSchema(conn, customerUser))

			restoreArgs = append(restoreArgs, fmt.Sprintf("--schema=%s", customerUser))
		}

		log("Restoring the dump file")
		conn.Username = customerUser
		abort(
			setup.RestoreDumpFile(conn, pDumpFile, setup.RestoreOptions{CustomArgs: restoreArgs}))

		log("Done.")
	},
}

func log(message ...string) {
	s := make([]interface{}, len(message)-1)
	for i := 1; i < len(message); i++ {
		s[i-1] = message[i]
	}
	fmt.Println(fmt.Sprintf(message[0], s...))
}

// abort: aborts this program on any error
func abort(err error) {
	if err != nil {
		panic(err)
	}
}

func checkInputParameters() bool {

	if pCustomerID == 0 {
		fmt.Println("Missing the customer id!")
		return false
	}

	if pDumpFile == "" {
		fmt.Println("Missing the dump file!")
		return false

	}

	return true
}

func cleanup(conn setup.ConnectionDetails, customerUser string) {
	if pCleanInstall {

		log("Cleaning up the '%s' database", pTargetDatabase)
		abort(
			setup.DropDatabase(conn, pTargetDatabase))
		for _, user := range []string{pTargetUserName, customerUser} {
			log("Dropping the '%s' user", user)
			abort(
				setup.DropUser(conn, user))
		}
	}
}

var (
	pCustomerID, pDBPort                                 int
	pDBUserName, pDBHost, pDBName, pDBSslMode, pDumpFile string
	pDBPassword                                          string
	pTargetDatabase, pTargetUserName                     string
	pCleanInstall                                        bool
)

func init() {
	RootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolVarP(&pCleanInstall, "force-cleanup", "", false, "Remove the database and user before starts (DANGER)")

	installCmd.Flags().StringVarP(&pDBSslMode, "ssl-mode", "S", "disable", "SSL connection: 'require', 'verify-full', 'verify-ca', and 'disable' supported")
	installCmd.Flags().IntVarP(&pCustomerID, "customer", "c", 0, "Your customer ID")
	installCmd.Flags().IntVarP(&pDBPort, "port", "p", 5432, "Database port")
	installCmd.Flags().StringVarP(&pDBUserName, "username", "U", "postgres", "Database user")
	installCmd.Flags().StringVarP(&pDBPassword, "password", "P", "", "Username password")
	installCmd.Flags().StringVarP(&pDBHost, "host", "", "127.0.0.1", "Database host")
	installCmd.Flags().StringVarP(&pDBName, "database", "d", "postgres", "Database name")
	installCmd.Flags().StringVarP(&pDumpFile, "dump-file", "D", "", "Database dump file")
	installCmd.Flags().StringVarP(&pTargetDatabase, "target-database", "", "umovme_dbview_db", "The target database")
	installCmd.Flags().StringVarP(&pTargetUserName, "target-username", "", "dbview", "The target username")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// installCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}