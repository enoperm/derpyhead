package cmd

import (
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"go4.org/mem"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/types/key"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		http.HandleFunc("/", respondPeerIds)
		os.Remove(appConfig.listenAddr)
		listenSock, err := net.Listen("unix", appConfig.listenAddr)
		if err != nil {
			log.Fatal(err)
		}
		defer listenSock.Close()
		log.Println(http.Serve(listenSock, nil))
	},
}

func init() {
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func respondPeerIds(w http.ResponseWriter, req *http.Request) {
	status, err := fetchStatus()
	if err != nil {
		log.Println(err)
		os.Stdout.WriteString(err.Error())
		return
	}
	encoder := json.NewEncoder(w)
	encoder.Encode(status)
}

func fetchStatus() (status *ipnstate.Status, err error) {
	db, err := sql.Open("sqlite3", appConfig.sourceDatabase)
	if err != nil {
		return
	}
	defer db.Close()

	sb := ipnstate.StatusBuilder{}
	rows, err := db.Query(`
		SELECT machines.node_key, namespaces.name AS namespace_name
		FROM machines, namespaces
		WHERE machines.namespace_id = namespaces.id
	`)

	switch {
	case err == sql.ErrNoRows:
		err = nil
		fallthrough
	case err != nil:
		return
	}

	type record struct {
		nodeKey   string
		namespace string
	}

	queue := make(chan record, 4)
	go func() {
		defer close(queue)
		for rows.Next() {
			var nodeKey, namespace string
			rows.Scan(&nodeKey, &namespace)
			if appConfig.includeRegex == nil || appConfig.includeRegex.MatchString(namespace) {
				queue <- record{
					nodeKey:   nodeKey,
					namespace: namespace,
				}
			}
		}
	}()

	dummyStatus := &ipnstate.PeerStatus{}
	for record := range queue {
		if appConfig.excludeRegex != nil && appConfig.excludeRegex.MatchString(record.namespace) {
			continue
		}

		nodeKey, err := key.NewPublicFromHexMem(mem.S(record.nodeKey))
		if err != nil {
			log.Println(err)
			continue
		}
		sb.AddPeer(nodeKey, dummyStatus)
	}

	status = sb.Status()
	var nullKey [32]byte

	selfKey, _ := key.NewPublicFromHexMem(mem.B(nullKey[:]))
	status.Self = &ipnstate.PeerStatus{
		PublicKey: selfKey,
	}
	return
}
