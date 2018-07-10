package stv

import (
	"fmt"
	"sort"
	"encoding/json"
	"github.com/dispatchlabs/samples/election_poc/types"
)

type Election struct {
	NbrVacancies		int64				`json:"nbrVacancies,omitempty"`
	Ballots				[]types.Ballot		`json:"-"`
	Droop				float64				`json:"droop,omitempty"`
	ElectionResults		*ElectionResults	`json:"electionResult,omitempty"`
	Elected				[]types.Candidate	`json:"elected,omitempty"`
	Hopefuls			[]*types.Candidate	`json:"hopefuls,omitempty"`
	Eliminated			[]types.Candidate	`json:"eliminated,omitempty"`
	CandidateMap		map[string]*types.Candidate `json:"-"`
}

func (this *Election) DoElection() {
	nbrBallots := float64(len(this.Ballots))
	denom := float64(this.NbrVacancies + 1)
	this.Droop =  float64(nbrBallots / denom) + 1

	fmt.Printf("Droop = %f\n", this.Droop)
	var roundNbr int64

	roundNbr = 1
	for int64(len(this.ElectionResults.ElectionResults)) < this.NbrVacancies {

		electionRound := this.ExecuteRound(roundNbr, this.Ballots)
		electedCandidates := make([]*ElectionResult, 0)

		for _, cand := range electionRound.Elected {
			electedCandidates = append(electedCandidates, &ElectionResult{Candidate: cand, TotalVotes: cand.CurrentVotes, RoundNumber: roundNbr, Result: types.StatusElected})
		}
		electionResults := &ElectionResults {
			ElectionResults: electedCandidates,
			Elected: electionRound.Elected,
			Eliminated: electionRound.Eliminated,
		}






		this.ReconcileMainRound(electionResults)

		sort.Sort(sort.Reverse(electionResults))
		remaining := this.NbrVacancies - int64(len(this.ElectionResults.Elected))
		if remaining <= 0 {
			fmt.Printf("\nEnough candidates are elected")
			break
		} else if(int64(len(electionResults.Elected)) > remaining) {
			this.ElectionResults.ElectionResults = append(this.ElectionResults.ElectionResults, electionResults.ElectionResults[:remaining]...)
			this.ElectionResults.Elected = append(this.ElectionResults.Elected, electionResults.Elected[:remaining]...)

		} else {
			this.ElectionResults.ElectionResults = append(this.ElectionResults.ElectionResults, electionResults.ElectionResults...)
			this.ElectionResults.Elected = append(this.ElectionResults.Elected, electionResults.Elected...)
		}
		this.ElectionResults.Eliminated = append(this.ElectionResults.Eliminated, electionResults.Eliminated...)
		this.Redistribute(electionResults.Elected, roundNbr)
		fmt.Printf(this.ElectionResults.ToPrettyJson())
		roundNbr++
	}
	//electionResult := ElectionResults{results}
	//sort.Sort(sort.Reverse(electionResult))
	//fmt.Printf(electionResult.ToPrettyJson())

}

func (this *Election) ExecuteRound(roundNumber int64, ballots []types.Ballot) *ElectionRound {
	for _, ballot := range this.Ballots {
		for _, vote := range ballot.Votes {
			if vote.Rank == roundNumber {
				if vote.Candidate.ElectionStatus == types.StatusHopefull {
					this.getCandidate(vote.Candidate.Name).AddVotes(1)
				}
			}
		}
	}
	voteCounts := make([]VoteCount, 0)
	for k, v := range this.CandidateMap {
		if v.ElectionStatus == types.StatusHopefull {
			vote := VoteCount{
				Count: v.CurrentVotes,
				Candidate: v,
			}
			voteCounts = append(voteCounts, vote)
			fmt.Printf("Current Hopefuls: %s :: %v\n", k, v.CurrentVotes)
		}
	}
	electionRound := &ElectionRound{VoteCount: voteCounts}
	sort.Sort(sort.Reverse(electionRound))
	electionRound.CountRound(this.Droop, roundNumber)

	return electionRound
}


func (this *Election) ReconcileMainRound(results *ElectionResults) {
	for _, cand := range results.Elected {
		this.getCandidate(cand.Name).SetStatus(types.StatusElected)
	}
	for _, cand := range results.Eliminated {
		this.getCandidate(cand.Name).SetStatus(types.StatusEliminated)
	}
	updatedHopefuls := make([]*types.Candidate, 0)
	for _, cand := range this.Hopefuls {
		if this.getCandidate(cand.Name).ElectionStatus == types.StatusHopefull {
			updatedHopefuls = append(updatedHopefuls, cand)
		}
	}
	this.Hopefuls = updatedHopefuls
}

func (this Election) ToPrettyJson() string {
	jsn, err := json.MarshalIndent(this, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(jsn)
}


func (this *Election) isElected(candidate *types.Candidate) bool {
	result := false
	for _, elected := range this.Elected {
		if elected.Name == candidate.Name {
			result = true
		}
	}
	return result
}

func (this *Election) isEliminated(candidate *types.Candidate) bool {
	result := false
	for _, elected := range this.Eliminated {
		if elected.Name == candidate.Name {
			result = true
		}
	}
	return result
}

func (this *Election) getCandidate(candidateName string) *types.Candidate {
	return this.CandidateMap[candidateName]
}

func (this *Election) PrintBallots() {
	for _, ballot := range this.Ballots {
		fmt.Printf("Ballot: %v\n", ballot.ToJson())
	}
}