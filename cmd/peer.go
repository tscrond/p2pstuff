/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/tscrond/p2pstuff/internal/p2pnode"

	"github.com/spf13/cobra"
)

// peerCmd represents the peer command
var peerCmd = &cobra.Command{
	Use:   "peer",
	Short: "Start a Peer node",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		rendezvous, _ := cmd.Flags().GetString("rendezvous")
		protocol, _ := cmd.Flags().GetString("proto")
		bootstrapMode, _ := cmd.Flags().GetString("bootstrap-mode")

		ctx := context.Background()

		node, err := p2pnode.NewNode(ctx, rendezvous, protocol, bootstrapMode)
		if err != nil {
			log.Println("Failed initializing node:", err)
		}

		if node != nil {
			fmt.Println("Node initialized with peer ID:", node.ID().String())
		}

		go node.DiscoverNodes(ctx)

		select {}
	},
}

func init() {
	rootCmd.AddCommand(peerCmd)

	peerCmd.PersistentFlags().String("rendezvous", "f9cbb0a0c6e5b443a27335e0efc4ff45ba89e2ece34cb5b06404125a3deda0b8", "rendezvous string")
	peerCmd.PersistentFlags().String("proto", "/bobaklabs/1.0.0", "protocol ID")
	peerCmd.PersistentFlags().String("bootstrap-mode", "libp2p-default", "custom bootstrap nodes or libp2p-provided (options: libp2p-default, custom)")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// peerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// peerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
