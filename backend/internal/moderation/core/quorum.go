package moderation

const DefaultQuorum = 2

type QuorumState struct {
	ApprovalCount  int
	RejectionCount int
	Reached        bool
	FinalDecision  ReviewDecision
	Status         ReportStatus
	Resolution     string
}

func EvaluateQuorum(reviews []ReportReview, quorum int) QuorumState {
	if quorum <= 0 {
		quorum = DefaultQuorum
	}

	state := QuorumState{Status: ReportStatusOpen}
	for _, review := range reviews {
		switch review.Decision {
		case ReviewDecisionApprove:
			state.ApprovalCount++
		case ReviewDecisionReject:
			state.RejectionCount++
		}
	}

	if len(reviews) > 0 {
		state.Status = ReportStatusInReview
	}
	if state.ApprovalCount >= quorum {
		state.Reached = true
		state.FinalDecision = ReviewDecisionApprove
		state.Status = ReportStatusResolved
		state.Resolution = "poll_hidden"
	}
	if state.RejectionCount >= quorum {
		state.Reached = true
		state.FinalDecision = ReviewDecisionReject
		state.Status = ReportStatusRejected
		state.Resolution = "report_rejected"
	}

	return state
}
