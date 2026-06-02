package service

import (
	"exchange_cdk/model"
	"testing"
)

type purchaseTaskSequenceAllocator interface {
	AllocateNextSequence(teamOwnerID, templateID uint) (uint, error)
}

func newPurchaseTaskSequenceAllocatorForTest() purchaseTaskSequenceAllocator {
	return lookupPurchaseTaskSequenceAllocatorForTest()
}

func lookupPurchaseTaskSequenceAllocatorForTest() purchaseTaskSequenceAllocator {
	return nil
}

func TestPurchaseTaskSequenceAllocatorPersistsSequencePerTeamOwnerAndTemplate(t *testing.T) {
	allocator := newPurchaseTaskSequenceAllocatorForTest()
	if allocator == nil {
		t.Fatalf("purchase task sequence allocator not implemented: expected persistent sequence allocation per team owner and template")
	}

	sequence1, err := allocator.AllocateNextSequence(10, 20)
	if err != nil {
		t.Fatalf("allocate first sequence: %v", err)
	}
	sequence2, err := allocator.AllocateNextSequence(10, 20)
	if err != nil {
		t.Fatalf("allocate second sequence: %v", err)
	}
	otherTemplateSequence, err := allocator.AllocateNextSequence(10, 21)
	if err != nil {
		t.Fatalf("allocate other template sequence: %v", err)
	}
	otherOwnerSequence, err := allocator.AllocateNextSequence(11, 20)
	if err != nil {
		t.Fatalf("allocate other owner sequence: %v", err)
	}

	first := model.PurchaseTask{TeamOwnerID: 10, TemplateID: 20, SequenceNo: sequence1}
	second := model.PurchaseTask{TeamOwnerID: 10, TemplateID: 20, SequenceNo: sequence2}
	otherTemplate := model.PurchaseTask{TeamOwnerID: 10, TemplateID: 21, SequenceNo: otherTemplateSequence}
	otherOwner := model.PurchaseTask{TeamOwnerID: 11, TemplateID: 20, SequenceNo: otherOwnerSequence}

	if second.SequenceNo != first.SequenceNo+1 {
		t.Fatalf("expected allocator to increment persisted sequence for same team owner/template: first=%d second=%d", first.SequenceNo, second.SequenceNo)
	}
	if otherTemplate.SequenceNo != 1 {
		t.Fatalf("expected allocator to start sequence at 1 for a different template, got %d", otherTemplate.SequenceNo)
	}
	if otherOwner.SequenceNo != 1 {
		t.Fatalf("expected allocator to start sequence at 1 for a different team owner, got %d", otherOwner.SequenceNo)
	}
}
