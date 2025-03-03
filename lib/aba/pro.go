// Package aba (Abacus), protein level
package aba

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"philosopher/lib/iso"
	"philosopher/lib/msg"
	"philosopher/lib/uti"

	"philosopher/lib/dat"
	"philosopher/lib/fil"
	"philosopher/lib/id"
	"philosopher/lib/met"
	"philosopher/lib/rep"
	"philosopher/lib/sys"

	"github.com/sirupsen/logrus"
)

// Create protein combined report
func proteinLevelAbacus(m met.Data, args []string) {

	var names []string
	var database dat.Base
	var datasets = make(map[string]rep.Evidence)

	var labels = make(map[string]string)

	// restore database
	database = dat.Base{}
	database.RestoreWithPath(args[0])

	// restoring combined file
	logrus.Info("Processing combined file")
	evidences := processProteinCombinedFile(m.Abacus, database)

	// recover all files
	logrus.Info("Restoring protein results")

	for _, i := range args {

		// restoring the database
		var e rep.Evidence
		e.RestoreGranularWithPath(i)

		// collect interact full file names
		files, _ := ioutil.ReadDir(i)
		for _, f := range files {
			if strings.Contains(f.Name(), "annotation") {
				var annot = fmt.Sprintf("%s%s%s", i, string(filepath.Separator), f.Name())

				file, e := os.Open(annot)
				if e != nil {
					msg.ReadFile(errors.New("cannot open annotation file"), "fatal")
				}
				defer file.Close()

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					names := strings.Fields(scanner.Text())

					name := i + " " + names[0]

					labels[name] = names[1]
				}

				if e = scanner.Err(); e != nil {
					msg.Custom(errors.New("the annotation file looks to be empty"), "error")
				}
			}
		}

		// collect project names
		prjName := i
		if strings.Contains(prjName, string(filepath.Separator)) {
			prjName = strings.Replace(filepath.Base(prjName), string(filepath.Separator), "", -1)
		}

		// unique list and map of datasets
		datasets[prjName] = e
		names = append(names, prjName)
	}

	sort.Strings(names)

	// If the name starts with CONTROL  or control then we put CONTROL (regardless of what follows after first '_')
	// If the name starts with something else, then we first determine, for each experiment, if the annotation
	// follows GENE_condition_replicate format (meaning there are two '_' in the name) or just GENE_replicate
	// format (meaning there is only one '_')

	// If two '_', then we put in the second row GENE_condition (i.e. remove the second _ and what follows)
	// If only one '_', then we put in the second row just GENE (i.e. remove the first _ and what follows after)

	var reprintLabels []string
	if m.Abacus.Reprint {
		for i := range names {
			if strings.Contains(strings.ToUpper(names[i]), "CONTROL") {
				//strings.Replace(names[i], "control", "CONTROL", 1)
				reprintLabels = append(reprintLabels, "CONTROL")
			} else {
				parts := strings.Split(names[i], "_")
				if len(parts) == 3 {
					label := fmt.Sprintf("%s_%s", parts[0], parts[1])
					reprintLabels = append(reprintLabels, label)
				} else if len(parts) == 2 {
					label := parts[0]
					reprintLabels = append(reprintLabels, label)
				}
			}
		}
	}

	sort.Strings(names)
	sort.Strings(reprintLabels)

	logrus.Info("Processing spectral counts")
	evidences = getProteinSpectralCounts(evidences, datasets, m.Abacus.Tag)

	logrus.Info("Processing peptide counts")
	evidences = getProteinToPeptideCounts(evidences, datasets, m.Abacus.Tag)

	logrus.Info("Processing intensities")
	evidences = sumProteinIntensities(evidences, datasets)

	// collect TMT labels
	if m.Abacus.Labels {
		evidences = getProteinLabelIntensities(evidences, datasets, m.Abacus.Tag)
	}

	if m.Abacus.Labels {
		saveProteinAbacusResult(m.Temp, m.Abacus.Plex, evidences, datasets, names, m.Abacus.Unique, true, m.Abacus.Full, labels)
	} else {
		saveProteinAbacusResult(m.Temp, m.Abacus.Plex, evidences, datasets, names, m.Abacus.Unique, false, m.Abacus.Full, labels)
	}

	if m.Abacus.Reprint {
		logrus.Info("Creating Reprint reports")
		saveReprintSpCResults(m.Temp, m.Abacus.Plex, evidences, datasets, names, reprintLabels, m.Abacus.Unique, false, labels)
		saveReprintIntResults(m.Temp, m.Abacus.Plex, evidences, datasets, names, reprintLabels, m.Abacus.Unique, false, labels)
	}

}

// processCombinedFile reads the combined protXML and creates a unique protein list as a reference fo all counts
func processProteinCombinedFile(a met.Abacus, database dat.Base) rep.CombinedProteinEvidenceList {

	var list rep.CombinedProteinEvidenceList

	if _, e := os.Stat("combined.prot.xml"); os.IsNotExist(e) {

		msg.Custom(errors.New("cannot find combined.prot.xml file"), "error")

	} else {

		var protxml id.ProtXML
		protxml.Read("combined.prot.xml")
		protxml.DecoyTag = a.Tag

		protxml.MarkUniquePeptides(1)

		// promote decoy proteins with indistinguishable target proteins
		protxml.PromoteProteinIDs()

		// applies pickedFDR algorithm
		if a.Picked {
			protxml = fil.PickedFDR(protxml)
		}

		// applies razor algorithm
		if a.Razor {
			protxml = fil.RazorFilter(protxml)
		}

		proid := fil.ProtXMLFilter(protxml, 0.01, a.PepProb, a.ProtProb, a.Picked, a.Razor, a.Tag)

		for _, j := range proid {

			if !strings.HasPrefix(j.ProteinName, a.Tag) {

				var ce rep.CombinedProteinEvidence

				ce.TotalSpc = make(map[string]int)
				ce.UniqueSpc = make(map[string]int)
				ce.UrazorSpc = make(map[string]int)

				ce.TotalPeptides = make(map[string]map[string]bool)
				ce.UniquePeptides = make(map[string]map[string]bool)
				ce.UrazorPeptides = make(map[string]map[string]bool)

				ce.TotalIntensity = make(map[string]float64)
				ce.UniqueIntensity = make(map[string]float64)
				ce.UrazorIntensity = make(map[string]float64)

				ce.TotalLabels = make(map[string]iso.Labels)
				ce.UniqueLabels = make(map[string]iso.Labels)
				ce.URazorLabels = make(map[string]iso.Labels)

				ce.SupportingSpectra = make(map[string]string)
				ce.ProteinName = j.ProteinName
				ce.Length = j.Length
				ce.Coverage = j.PercentCoverage
				ce.GroupNumber = j.GroupNumber
				ce.SiblingID = j.GroupSiblingID
				ce.IndiProtein = j.IndistinguishableProtein
				ce.UniqueStrippedPeptides = 0
				ce.PeptideIons = j.PeptideIons
				ce.ProteinProbability = j.Probability
				ce.TopPepProb = j.TopPepProb

				list = append(list, ce)
			}
		}

	}

	for i := range list {
		for _, j := range database.Records {
			if strings.Contains(j.OriginalHeader, list[i].ProteinName) && strings.HasPrefix(j.OriginalHeader, list[i].ProteinID) && !strings.Contains(j.OriginalHeader, a.Tag) {
				//if strings.Contains(j.OriginalHeader, list[i].ProteinName) && !strings.Contains(j.OriginalHeader, a.Tag) {
				list[i].ProteinName = j.PartHeader
				list[i].ProteinID = j.ID
				list[i].EntryName = j.EntryName
				list[i].GeneNames = j.GeneNames
				list[i].Organism = j.Organism
				list[i].Description = j.Description
				list[i].ProteinExistence = j.ProteinExistence
				break
			}
		}
	}

	return list
}

// getProteinSpectralCounts collects protein spectral counts from the individual data sets for the combined protein report
func getProteinSpectralCounts(combined rep.CombinedProteinEvidenceList, datasets map[string]rep.Evidence, decoyTag string) rep.CombinedProteinEvidenceList {

	for i := range combined {
		for k, v := range datasets {
			for _, j := range v.Proteins {
				if combined[i].ProteinID == j.ProteinID && !strings.Contains(j.OriginalHeader, decoyTag) {
					combined[i].UniqueSpc[k] = j.UniqueSpC
					combined[i].TotalSpc[k] = j.TotalSpC
					combined[i].UrazorSpc[k] = j.URazorSpC
					break
				}
			}
		}
	}

	return combined
}

// // getProteinToPeptideCounts collects peptide counts from the individual data sets for the combined protein report
// getProteinToPeptideCounts collects peptide counts from the individual data sets for the combined protein report
func getProteinToPeptideCounts(combined rep.CombinedProteinEvidenceList, datasets map[string]rep.Evidence, decoyTag string) rep.CombinedProteinEvidenceList {

	for i := range combined {

		var total []string
		var unique []string
		var razor []string

		for k, v := range datasets {
			for _, j := range v.Proteins {
				if combined[i].ProteinName == j.PartHeader && !strings.Contains(j.OriginalHeader, decoyTag) {

					for l := range j.TotalPeptides {
						total = append(total, l)
					}

					for l := range j.UniquePeptides {
						unique = append(unique, l)
					}

					for l := range j.URazorPeptides {
						razor = append(razor, l)
					}
				}
			}

			total = uti.RemoveDuplicateStrings(total)
			var totalMap = make(map[string]bool)
			for _, k := range total {
				totalMap[k] = false
			}
			combined[i].TotalPeptides[k] = totalMap

			unique = uti.RemoveDuplicateStrings(unique)
			var uniqueMap = make(map[string]bool)
			for _, k := range unique {
				uniqueMap[k] = false
			}
			combined[i].UniquePeptides[k] = uniqueMap

			razor = uti.RemoveDuplicateStrings(razor)
			var razorMap = make(map[string]bool)
			for _, k := range razor {
				razorMap[k] = false
			}
			combined[i].UrazorPeptides[k] = razorMap
		}
	}

	return combined
}

// getProteinLabelIntensities collects protein isobaric quantification from the individual data sets for the combined protein report
func getProteinLabelIntensities(combined rep.CombinedProteinEvidenceList, datasets map[string]rep.Evidence, decoyTag string) rep.CombinedProteinEvidenceList {

	for k, v := range datasets {

		for i := range combined {
			for _, j := range v.Proteins {
				if combined[i].ProteinID == j.ProteinID && !strings.Contains(j.OriginalHeader, decoyTag) {
					combined[i].TotalLabels[k] = *j.TotalLabels
					combined[i].UniqueLabels[k] = *j.UniqueLabels
					combined[i].URazorLabels[k] = *j.URazorLabels
					break
				}
			}
		}

	}

	return combined
}

// sumIntensities calculates the protein intensity
func sumProteinIntensities(combined rep.CombinedProteinEvidenceList, datasets map[string]rep.Evidence) rep.CombinedProteinEvidenceList {

	for k, v := range datasets {

		var ions = make(map[id.IonFormType]float64)
		for _, i := range v.Ions {
			ions[i.IonForm()] = i.Intensity
		}

		for _, i := range combined {
			for j := range v.Proteins {
				if i.ProteinID == v.Proteins[j].ProteinID {
					i.TotalIntensity[k] = v.Proteins[j].TotalIntensity
					i.UniqueIntensity[k] = v.Proteins[j].UniqueIntensity
					i.UrazorIntensity[k] = v.Proteins[j].URazorIntensity
					break
				}
			}
		}

	}

	return combined
}

// saveProteinAbacusResult creates a single report using 1 or more philosopher result files
func saveProteinAbacusResult(session, plex string, evidences rep.CombinedProteinEvidenceList, datasets map[string]rep.Evidence, namesList []string, uniqueOnly, hasLabels, full bool, labelsList map[string]string) {

	var summTotalSpC = make(map[string]int)
	var summUniqueSpC = make(map[string]int)
	var summURazorSpC = make(map[string]int)

	var totalPeptides = make(map[string][]string)
	var uniquePeptides = make(map[string][]string)
	var razorPeptides = make(map[string][]string)

	// organize by group number
	sort.Sort(evidences)

	// collect and sum all evidences from all data sets for each protein
	for _, i := range evidences {
		for _, j := range namesList {
			summTotalSpC[i.ProteinID] += i.TotalSpc[j]
			summUniqueSpC[i.ProteinID] += i.UniqueSpc[j]
			summURazorSpC[i.ProteinID] += i.UrazorSpc[j]

			totalPeptideMap := i.TotalPeptides[j]
			for k := range totalPeptideMap {
				totalPeptides[i.ProteinID] = append(totalPeptides[i.ProteinID], k)
			}

			uniquePeptideMap := i.UniquePeptides[j]
			for k := range uniquePeptideMap {
				uniquePeptides[i.ProteinID] = append(uniquePeptides[i.ProteinID], k)
			}

			razorPeptideMap := i.UrazorPeptides[j]
			for k := range razorPeptideMap {
				razorPeptides[i.ProteinID] = append(razorPeptides[i.ProteinID], k)
			}
		}

		totalPeptides[i.ProteinID] = uti.RemoveDuplicateStrings(totalPeptides[i.ProteinID])
		uniquePeptides[i.ProteinID] = uti.RemoveDuplicateStrings(uniquePeptides[i.ProteinID])
		razorPeptides[i.ProteinID] = uti.RemoveDuplicateStrings(razorPeptides[i.ProteinID])
	}

	// create result file
	output := fmt.Sprintf("%s%scombined_protein.tsv", session, string(filepath.Separator))

	// create result file
	file, e := os.Create(output)
	if e != nil {
		msg.WriteFile(e, "fatal")
	}
	defer file.Close()

	header := "Protein\tProtein ID\tEntry Name\tGene\tProtein Length\tOrganism\tProtein Existence\tDescription\tProtein Probability\tTop Peptide Probability\tCombined Total Peptides\tCombined Spectral Count\tCombined Unique Spectral Count\tCombined Total Spectral Count"

	// Add Unique+Razor SPC
	for _, i := range namesList {
		header += fmt.Sprintf("\t%s Spectral Count", i)
	}

	// Add Unique SPC
	if full {
		for _, i := range namesList {
			header += fmt.Sprintf("\t%s Unique Spectral Count", i)
		}
	}

	// Add Total SPC
	if full {
		for _, i := range namesList {
			header += fmt.Sprintf("\t%s Total Spectral Count", i)
		}
	}

	// Add Unique+Razor Intensity
	for _, i := range namesList {
		header += fmt.Sprintf("\t%s Intensity", i)
	}

	// Add Unique Intensity
	if full {
		for _, i := range namesList {
			header += fmt.Sprintf("\t%s Unique Intensity", i)
		}
	}

	// Add Total Intensity
	if full {
		for _, i := range namesList {
			header += fmt.Sprintf("\t%s Total Intensity", i)
		}
	}

	var chs []string

	if plex == "10" {
		chs = append(chs, "126", "127N", "127C", "128N", "128C", "129N", "129C", "130N", "130C", "131N")
	} else if plex == "11" {
		chs = append(chs, "126", "127N", "127C", "128N", "128C", "129N", "129C", "130N", "130C", "131N", "131C")
	} else if plex == "16" {
		chs = append(chs, "126", "127N", "127C", "128N", "128C", "129N", "129C", "130N", "130C", "131N", "131C", "132N", "132C", "133N", "133C", "134N")
	} else if plex == "18" {
		chs = append(chs, "126", "127N", "127C", "128N", "128C", "129N", "129C", "130N", "130C", "131N", "131C", "132N", "132C", "133N", "133C", "134N", "134C", "135N")
	} else {
		msg.Custom(errors.New("unsupported number of labels"), "error")
	}

	if hasLabels {
		for _, i := range namesList {
			for _, j := range chs {
				l := fmt.Sprintf("%s %s", i, j)
				v, ok := labelsList[l]
				if ok {
					header += fmt.Sprintf("\t%s", v)
				} else {
					header += fmt.Sprintf("\t%s %s", i, j)
				}
			}
		}
	}

	header += "\tIndistinguishable Proteins"

	header += "\n"
	_, e = io.WriteString(file, header)
	if e != nil {
		msg.WriteToFile(e, "error")
	}

	for _, i := range evidences {

		if len(i.TotalSpc) > 0 {

			var line string

			line += fmt.Sprintf("%s\t", i.ProteinName)

			line += fmt.Sprintf("%s\t", i.ProteinID)

			line += fmt.Sprintf("%s\t", i.EntryName)

			line += fmt.Sprintf("%s\t", i.GeneNames)

			line += fmt.Sprintf("%d\t", i.Length)

			line += fmt.Sprintf("%s\t", i.Organism)

			line += fmt.Sprintf("%s\t", i.ProteinExistence)

			line += fmt.Sprintf("%s\t", i.Description)

			line += fmt.Sprintf("%.4f\t", i.ProteinProbability)

			line += fmt.Sprintf("%.4f\t", i.TopPepProb)

			line += fmt.Sprintf("%d\t", len(totalPeptides[i.ProteinID]))

			line += fmt.Sprintf("%d\t", summURazorSpC[i.ProteinID])

			line += fmt.Sprintf("%d\t", summUniqueSpC[i.ProteinID])

			line += fmt.Sprintf("%d\t", summTotalSpC[i.ProteinID])

			// Add Unique+Razor SPC
			for _, j := range namesList {
				line += fmt.Sprintf("%d\t", i.UrazorSpc[j])
			}

			// Add Unique SPC
			if full {
				for _, j := range namesList {
					line += fmt.Sprintf("%d\t", i.UniqueSpc[j])
				}
			}

			// Add Total SPC
			if full {
				for _, j := range namesList {
					line += fmt.Sprintf("%d\t", i.TotalSpc[j])
				}
			}

			// Add Unique+Razor Int
			for _, j := range namesList {
				line += fmt.Sprintf("%6.f\t", i.UrazorIntensity[j])
			}

			// Add Unique Int
			if full {
				for _, j := range namesList {
					line += fmt.Sprintf("%6.f\t", i.UniqueIntensity[j])
				}
			}

			// Add Total Int
			if full {
				for _, j := range namesList {
					line += fmt.Sprintf("%6.f\t", i.TotalIntensity[j])
				}
			}

			if hasLabels {
				switch plex {
				case "10":
					for _, j := range namesList {
						line += fmt.Sprintf("%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t",
							i.URazorLabels[j].Channel1.Intensity,
							i.URazorLabels[j].Channel2.Intensity,
							i.URazorLabels[j].Channel3.Intensity,
							i.URazorLabels[j].Channel4.Intensity,
							i.URazorLabels[j].Channel5.Intensity,
							i.URazorLabels[j].Channel6.Intensity,
							i.URazorLabels[j].Channel7.Intensity,
							i.URazorLabels[j].Channel8.Intensity,
							i.URazorLabels[j].Channel9.Intensity,
							i.URazorLabels[j].Channel10.Intensity,
						)
					}
				case "16":
					for _, j := range namesList {
						line += fmt.Sprintf("%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t",
							i.URazorLabels[j].Channel1.Intensity,
							i.URazorLabels[j].Channel2.Intensity,
							i.URazorLabels[j].Channel3.Intensity,
							i.URazorLabels[j].Channel4.Intensity,
							i.URazorLabels[j].Channel5.Intensity,
							i.URazorLabels[j].Channel6.Intensity,
							i.URazorLabels[j].Channel7.Intensity,
							i.URazorLabels[j].Channel8.Intensity,
							i.URazorLabels[j].Channel9.Intensity,
							i.URazorLabels[j].Channel10.Intensity,
							i.URazorLabels[j].Channel11.Intensity,
							i.URazorLabels[j].Channel12.Intensity,
							i.URazorLabels[j].Channel13.Intensity,
							i.URazorLabels[j].Channel14.Intensity,
							i.URazorLabels[j].Channel15.Intensity,
							i.URazorLabels[j].Channel16.Intensity,
						)
					}
				case "18":
					for _, j := range namesList {
						line += fmt.Sprintf("%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t",
							i.URazorLabels[j].Channel1.Intensity,
							i.URazorLabels[j].Channel2.Intensity,
							i.URazorLabels[j].Channel3.Intensity,
							i.URazorLabels[j].Channel4.Intensity,
							i.URazorLabels[j].Channel5.Intensity,
							i.URazorLabels[j].Channel6.Intensity,
							i.URazorLabels[j].Channel7.Intensity,
							i.URazorLabels[j].Channel8.Intensity,
							i.URazorLabels[j].Channel9.Intensity,
							i.URazorLabels[j].Channel10.Intensity,
							i.URazorLabels[j].Channel11.Intensity,
							i.URazorLabels[j].Channel12.Intensity,
							i.URazorLabels[j].Channel13.Intensity,
							i.URazorLabels[j].Channel14.Intensity,
							i.URazorLabels[j].Channel15.Intensity,
							i.URazorLabels[j].Channel16.Intensity,
							i.URazorLabels[j].Channel17.Intensity,
							i.URazorLabels[j].Channel18.Intensity,
						)
					}
				}
			}

			ip := strings.Join(i.IndiProtein, ", ")
			line += fmt.Sprintf("%s\t", ip)

			line += "\n"
			_, e := io.WriteString(file, line)
			if e != nil {
				msg.WriteToFile(e, "error")
			}

		}
	}

	// copy to work directory
	sys.CopyFile(output, filepath.Base(output))

}

// saveReprintSpCResults creates a single Spectral Count report using 1 or more philosopher result files using the Reprint format
func saveReprintSpCResults(session, plex string, evidences rep.CombinedProteinEvidenceList, datasets map[string]rep.Evidence, namesList, labelList []string, uniqueOnly, hasTMT bool, labelsList map[string]string) {

	// create result file
	output := fmt.Sprintf("%s%sreprint.spc.tsv", session, string(filepath.Separator))

	// create result file
	file, e := os.Create(output)
	if e != nil {
		msg.WriteFile(errors.New("cannot create reprint SpC report"), "fatal")
	}
	defer file.Close()

	line := "PROTID\tGENEID\tPROTLEN\t"

	for _, i := range namesList {
		line += fmt.Sprintf("%s_SPC\t", i)
	}

	line += "\n"
	line += "na\tna\tna\t"

	for _, i := range labelList {
		line += fmt.Sprintf("%s\t", i)
	}

	line += "\n"

	_, e = io.WriteString(file, line)
	if e != nil {
		msg.WriteToFile(e, "error")
	}

	// organize by group number
	sort.Sort(evidences)

	for _, i := range evidences {

		var line string

		line += fmt.Sprintf("%s\t%s\t%d\t", i.ProteinID, i.GeneNames, i.Length)

		for _, j := range namesList {
			line += fmt.Sprintf("%d\t", i.UrazorSpc[j])
		}

		line += "\n"
		_, e := io.WriteString(file, line)
		if e != nil {
			msg.WriteToFile(e, "error")
		}

	}

	// copy to work directory
	sys.CopyFile(output, filepath.Base(output))

}

// saveReprintIntResults creates a single Intensity-based report using 1 or more philosopher result files using the Reprint format
func saveReprintIntResults(session, plex string, evidences rep.CombinedProteinEvidenceList, datasets map[string]rep.Evidence, namesList, labelList []string, uniqueOnly, hasTMT bool, labelsList map[string]string) {

	// create result file
	output := fmt.Sprintf("%s%sreprint.int.tsv", session, string(filepath.Separator))

	// create result file
	file, e := os.Create(output)
	if e != nil {
		msg.WriteFile(errors.New("cannot create reprint Int. report"), "fatal")
	}
	defer file.Close()

	line := "PROTID\tGENEID\t"

	for _, i := range namesList {
		line += fmt.Sprintf("%s_INT\t", i)
	}

	line += "\n"
	line += "na\tna\t"

	for _, i := range labelList {
		line += fmt.Sprintf("%s\t", i)
	}

	line += "\n"

	_, e = io.WriteString(file, line)
	if e != nil {
		msg.WriteToFile(e, "error")
	}

	// organize by group number
	sort.Sort(evidences)

	for _, i := range evidences {

		var line string

		line += fmt.Sprintf("%s\t%s\t", i.ProteinID, i.GeneNames)

		for _, j := range namesList {
			line += fmt.Sprintf("%f\t", i.UrazorIntensity[j])
		}

		line += "\n"
		_, e := io.WriteString(file, line)
		if e != nil {
			msg.WriteToFile(e, "error")
		}

	}

	// copy to work directory
	sys.CopyFile(output, filepath.Base(output))
}
