package main

import (
	"encoding/json"
	"fmt"
)

type BeadsAdapter struct {
	runner interface {
		Run(name string, args ...string) ([]byte, error)
	}
}

type Issue struct {
	ID                 string
	Title              string
	Description        string
	AcceptanceCriteria string
}

type ReadyChild struct {
	ID        string
	IssueType string
	Status    string
	Priority  int
}

func NewBeadsAdapter(runner interface {
	Run(name string, args ...string) ([]byte, error)
}) *BeadsAdapter {
	return &BeadsAdapter{runner: runner}
}

func (b *BeadsAdapter) Show(id string) (Issue, error) {
	output, err := b.runner.Run("bd", "show", id, "--json")
	if err != nil {
		return Issue{}, err
	}

	var payload []struct {
		ID                 string `json:"id"`
		Title              string `json:"title"`
		Description        string `json:"description"`
		AcceptanceCriteria string `json:"acceptance_criteria"`
	}

	if err := json.Unmarshal(output, &payload); err != nil {
		return Issue{}, err
	}
	if len(payload) == 0 {
		return Issue{}, fmt.Errorf("bd show returned empty array")
	}

	item := payload[0]
	return Issue{
		ID:                 item.ID,
		Title:              item.Title,
		Description:        item.Description,
		AcceptanceCriteria: item.AcceptanceCriteria,
	}, nil
}

func (b *BeadsAdapter) ReadyChildren(parentID string) ([]ReadyChild, error) {
	output, err := b.runner.Run("bd", "ready", "--parent", parentID, "--json")
	if err != nil {
		return nil, err
	}

	var payload []struct {
		ID        string `json:"id"`
		IssueType string `json:"issue_type"`
		Status    string `json:"status"`
		Priority  int    `json:"priority"`
	}

	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, err
	}

	items := make([]ReadyChild, 0, len(payload))
	for _, item := range payload {
		items = append(items, ReadyChild{
			ID:        item.ID,
			IssueType: item.IssueType,
			Status:    item.Status,
			Priority:  item.Priority,
		})
	}

	return items, nil
}

func (b *BeadsAdapter) UpdateStatus(id string, status string) error {
	_, err := b.runner.Run("bd", "update", id, "--status", status)
	return err
}

func (b *BeadsAdapter) Close(id string) error {
	_, err := b.runner.Run("bd", "close", id)
	return err
}

func (b *BeadsAdapter) Sync() error {
	_, err := b.runner.Run("bd", "sync")
	return err
}
