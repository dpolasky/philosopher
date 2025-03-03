// Package cmd Abacus top level command
package cmd

import (
	"errors"
	"os"

	"philosopher/lib/aba"
	"philosopher/lib/met"
	"philosopher/lib/msg"
	"philosopher/lib/sys"

	"github.com/spf13/cobra"
)

// abacusCmd represents the abacus command
var abacusCmd = &cobra.Command{
	Use:   "abacus",
	Short: "Combined analysis of LC-MS/MS results",
	Run: func(cmd *cobra.Command, args []string) {

		m.FunctionInitCheckUp()

		if len(args) < 2 {
			msg.InputNotFound(errors.New("the combined analysis needs at least 2 result files to work"), "fatal")
		}

		msg.Executing("Abacus", Version)
		aba.Run(m, args)

		// store parameters on meta data
		m.Serialize()

		// clean tmp
		met.CleanTemp(m.Temp)

		msg.Done()
	},
}

func init() {

	if len(os.Args) > 1 && os.Args[1] == "abacus" {

		m.Restore(sys.Meta())

		abacusCmd.Flags().StringVarP(&m.Abacus.Tag, "tag", "", "rev_", "decoy tag")
		abacusCmd.Flags().StringVarP(&m.Abacus.Plex, "plex", "", "10", "number of channels")
		abacusCmd.Flags().Float64VarP(&m.Abacus.ProtProb, "prtProb", "", 0.9, "minimum protein probability")
		abacusCmd.Flags().Float64VarP(&m.Abacus.PepProb, "pepProb", "", 0.5, "minimum peptide probability")
		abacusCmd.Flags().BoolVarP(&m.Abacus.Protein, "protein", "", false, "global level protein report")
		abacusCmd.Flags().BoolVarP(&m.Abacus.Peptide, "peptide", "", false, "global level peptide report")
		abacusCmd.Flags().BoolVarP(&m.Abacus.Razor, "razor", "", false, "use razor peptides for protein FDR scoring")
		abacusCmd.Flags().BoolVarP(&m.Abacus.Picked, "picked", "", false, "apply the picked FDR algorithm before the protein scoring")
		abacusCmd.Flags().BoolVarP(&m.Abacus.Unique, "uniqueonly", "", false, "report TMT quantification based on only unique peptides")
		abacusCmd.Flags().BoolVarP(&m.Abacus.Labels, "labels", "", false, "indicates whether the data sets includes TMT labels or not")
		abacusCmd.Flags().BoolVarP(&m.Abacus.Reprint, "reprint", "", false, "create abacus reports using the Reprint format")
		abacusCmd.Flags().BoolVarP(&m.Abacus.Full, "full", "", false, "generates combined tables with extra information")
	}

	RootCmd.AddCommand(abacusCmd)
}
