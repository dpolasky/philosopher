package wmm

import (
	"fmt"
	"os"
	"philosopher/lib/met"
)

// Run executes the Filter processing
func Run(m met.Data) met.Data {

	var text string

	// Database
	text = writeDatabase(m.Database, text)

	// MSFragger
	if len(m.MSFragger.JarPath) > 0 {
		text = writeMSFragger(m.MSFragger, text)
	}

	// Philosopher
	//text = writePhilosopher(m.Filter, m.Quantify, text)

	f, _ := os.Create("methods.txt")
	defer f.Close()

	f.WriteString(text)

	return m
}

func writeDatabase(d met.Database, text string) string {

	var dbFlavor string
	if d.Rev {
		dbFlavor = "Swiss-Prot"
	} else {
		dbFlavor = "TrEMBL"
	}

	text = fmt.Sprintf("A protein database file was downloaded from UniProt %s (PMID:30395287) using the proteome ID %s on %s.", dbFlavor, d.ID, d.TimeStamp)

	if d.Crap {
		text = fmt.Sprintf("%s A list of 153 common contaminants was also added to the database.", text)
	}

	text = fmt.Sprintf("%s Decoy entries were generated by reversing the protein sequences and adding the %s prefix to their headers.", text, d.Tag)

	// appending new line before returning
	text = text + "\n"

	return text
}

func writeMSFragger(d met.MSFragger, text string) string {

	/* MSFRAGGER
	For the analysis of proteome data, MS/MS spectra were searched using a precursor-ion mass tolerance of %d ppm, fragment mass tolerance of %d ppm, and allowing C12/C13
	isotope errors %s. Cysteine carbamidomethylation (+57.0215) and lysine TMT labeling (+229.1629) were specified as fixed modifications, and methionine oxidation (+15.9949),
	N-terminal protein acetylation (+42.0106), and TMT labeling of peptide N terminus and serine residues were specified as variable modifications. The search was restricted
	to fully tryptic peptides, allowing up to two missed cleavage sites. For the analysis of phosphopeptide enriched data, the set of variable modifications also included
	phosphorylation (+79.9663) of serine, threonine, and tyrosine residues
	*/

	searchText := fmt.Sprintf("Database searching was performed on %s files with MSFragger [CITATION] using a precursor tolerance of", d.RawExtension)

	text = text + searchText

	return text
}

func writePhilosopher(f met.Filter, q met.Quantify, text string) string {

	/* PHILOSOPHER
	The search results were further processed using the Philosopher pipeline (https://github.com/Nesvilab/philosopher). First, MSFragger output files (in pepXML format)
	were processed using PeptideProphet (PMID:12403597) (with the high–mass accuracy binning and semi-parametric mixture modeling options) to compute the posterior probability
	of correct identification for each peptide to spectrum match (PSM). The resulting pepXML files from PeptideProphet were then processed together to assemble peptides
	into proteins (protein inference) and to create a combined file (in protXML format) of high confidence protein groups. Corresponding peptides were assigned to each group.
	The combined protXML file and the individual PSM lists for each TMT 10-plex were further processed using Philosopher filter command as follows. Each peptide was assigned
	either as a unique peptide to a particular protein group or assigned as a razor peptide to a single protein group that had the most peptide evidence. The protein groups
	assembled by ProteinProphet (Nesvizhskii et al., 2003) were filtered to 1% protein-level False Discovery Rate (FDR) using the chosen FDR target-decoy strategy and the best
	peptide approach (allowing both unique and razor peptides) and applying the picked FDR strategy (Savitski et al., 2015). In each TMT 10-plex, the PSM lists were filtered
	using a stringent, sequential FDR strategy, retaining only those PSMs with PeptideProphet probability of 0.9 or higher (which in these data corresponded to less than 1% PSM-level FDR)
	and mapped to proteins that also passed the global 1% protein-level FDR filter. For each PSM that passed these filters, MS1 intensity of the corresponding precursor-ion
	was extracted using the Philosopher label-free quantification module based on the moFF method (Argentini et al., 2016) (using 10 p.p.m mass tolerance and 0.4 min retention
	time window for extracted ion chromatogram peak tracing). In addition, for all PSMs corresponding to a TMT-labeled peptide, ten TMT reporter ion intensities were extracted from the MS/MS
	scans (using 0.002 Da window) and the precursor ion purity scores were calculated using the intensity of the sequenced precursor ion and that of other interfering ions
	observed in MS1 data (within a 0.7 Da isolation window).
	*/

	text = fmt.Sprintf("")

	return text
}

/* TMT-INTEGRATOR
All supporting information for each PSM,
including the accession numbers and names of the protein/gene selected based on the protein inference approach with razor peptide
assignment and quantification information (MS1 precursor-ion intensity and the TMT reporter ion intensities) was summarized in the
output PSM.tsv files, one file for each TMT 10-plex experiment. The PSM.tsv files were further processed using TMT-Integrator
(https://github.com/Nesvilab/TMT-Integrator) to generate summary reports at the gene and protein level and, for phosphopeptide
enriched data, also at the peptide and modification site levels. In the quantitation step, TMT-Integrator used as input the PSM tables
generated by the Philosopher pipeline as described above and created integrated reports with quantification across all samples at
each level. First, PSM from PSM.tsv files were filtered to remove all entries that did not pass at least one of the quality filters, such as
PSMs with (a) no TMT label; (b) missing quantification in the Reference sample; (c) precursor-ion purity less than 50%; (d) summed
reporter ion intensity (across all ten channels) in the lower 5% percentile of all PSMs in the corresponding PSM.tsv file (2.5% for phosphopeptide enriched data); (e) peptides without phosphorylation (for phosphopeptide enriched data). In the case of redundant PSMs
(i.e., multiple PSMs in the same MS run sample corresponding the same peptide ion), only the single PSM with the highest summed
TMT intensity was retained for subsequent analysis. Both unique and razor peptides were used for quantification, while PSMs mapping to common external contaminant proteins (that were included in the searched protein sequence database) were excluded. Next,
in each TMT 10-plex experiment, for each PSM the intensity in each TMT channel was log2 transformed, and the reference channel
intensity (pooled reference sample) was subtracted from that for the other nine channels (samples), thus converting the data into
log2-based ratio to the reference scale (referred to as ‘ratios’ below). After the ratio-to-reference conversion, the PSMs were grouped
on the basis of a predefined level (gene, protein, and also peptide and site-level for phosphopeptide enriched data; see below for
details). At each level, and in each sample, the interquartile range (IQR) algorithm was applied to remove the outliers in the
corresponding PSM group. The first quantile (Q1), the third quantile (Q3), and the interquartile range (IQR, i.e., Q3-Q1) of the sample
ratios were calculated, and the PSMs with ratios outside of the boundaries of Q1-1.5*IQR and Q3+1.5*IQR were excluded. Then, the
median was calculated from the remaining ratios to represent the ratio for each sample, at every level. In the next step, the ratios were
normalized using the median absolute deviation (MAD). Briefly, independently at each level of data summarization (gene, protein,
peptide, or site), given the p by n table of ratios for entry j in sample i, Rij, the median ratio Mi = median(Rij, j = 1,.,p), and the global
median across all n samples, M0 = median(Mi
, i = 1,.,n), were calculated. The ratios in each sample were median centered, RC
ij = Rij –
Mi
. The median absolute deviation of centered values in each sample, MADi = median(abs(RC
ij), j = 1.p) was calculated along with
the global absolute deviation, MAD0 = median(MADi
, i = 1,.,n). All ratios were then scaled to derive the final normalized measures:
RN
ij = (RC
ij/ MADi
) 3 MAD0 + M0. As a final step, the normalized ratios were converted back to the absolute intensity scale using the
estimated intensity of each entry (at each level, gene/protein/peptide/site) in the Reference sample. The Reference Intensity of entry i
measured in TMT 10-plex k (k = 1,.,q), REFik, was estimated using the weighted sum of the MS1 intensities of the top three most
intense peptide ions (Ning et al., 2012) quantified for that entry in the TMT 10-plex k. The weighting factor for each PSM was taken as
the proportion of the reference channel TMT intensity to the total summed TMT channel intensity. The overall Reference Intensity for
e9 Cell 179, 964–983.e1–e21, October 31, 2019
entry i was then computed as REFi = Mean(REFik, k = 1,.,q). In doing so, the missing intensity values (i.e., no identified and/or
quantified PSMs in a particular TMT 10-plex experiment) were imputed with a global minimum intensity value. The final abundance
(intensity) of entry i in sample j (log2 transformed) was computed as Aij = RN
ij + log2(REFi
). The ratio and intensity tables described
above were calculated separately for each level (gene and protein for whole proteome, and also peptide and site-level for phosphopeptide enriched data). PSMs were grouped as follows. At the gene level, all PSMs were grouped based on the gene symbol of the
corresponding protein to which they were assigned as either unique or razor peptides. In the protein tables, identified proteins that
mapped to the same gene were kept as separate entries.
*/

// The tutorial describing all steps of the analysis, including specific input parameter files, command-line option, and all software tools necessary to replicate the results are available at https://github.com/Nesvilab.//
