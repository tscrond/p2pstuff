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

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Start a bootstrap node",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		bnode, err := p2pnode.NewBootstrapNode(ctx)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println("Bootstrap node started!")
		fmt.Println("Peer ID:", bnode.ID())

		for _, addr := range bnode.Addrs() {
			fmt.Println("Listening on:", addr)
		}

		select {}
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// bootstrapCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// bootstrapCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
