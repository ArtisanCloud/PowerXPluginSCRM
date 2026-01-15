package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
)

// ErrPendingApprovalExists 当目标已有待审批记录时返回。
var ErrPendingApprovalExists = errors.New("pending approval already exists for target")

// ErrDecisionInvalid 表示审批决策缺失关键字段。
var ErrDecisionInvalid = errors.New("invalid approval decision")

// ApprovalHook 在审批状态更新后触发。
type ApprovalHook func(ctx context.Context, approval *model.ChangeApproval)

// ApprovalService 提供配置审批的业务能力。
type ApprovalService struct {
	repo   *repo.ApprovalRepository
	logger *logrus.Entry
	hooks  []ApprovalHook
}

// NewApprovalService 构造审批服务。
func NewApprovalService(repository *repo.ApprovalRepository, logger *logrus.Entry, hooks ...ApprovalHook) *ApprovalService {
	svc := &ApprovalService{
		repo:   repository,
		logger: logger,
	}
	for _, h := range hooks {
		if h != nil {
			svc.hooks = append(svc.hooks, h)
		}
	}
	return svc
}

// RegisterHook 动态注册审批回调。
func (s *ApprovalService) RegisterHook(h ApprovalHook) {
	if h != nil {
		s.hooks = append(s.hooks, h)
	}
}

// SubmitChangeRequest 请求创建新的审批。
type SubmitChangeRequest struct {
	TargetType  string
	TargetID    string
	Payload     any
	SubmittedBy string
}

// SubmitChange 创建新的待审批记录；若目标已有待处理请求会返回 ErrPendingApprovalExists。
func (s *ApprovalService) SubmitChange(ctx context.Context, req SubmitChangeRequest) (*model.ChangeApproval, error) {
	if strings.TrimSpace(req.TargetType) == "" || strings.TrimSpace(req.TargetID) == "" {
		return nil, fmt.Errorf("target_type and target_id are required: %w", repo.ErrApprovalInvalid)
	}
	if strings.TrimSpace(req.SubmittedBy) == "" {
		return nil, fmt.Errorf("submitted_by is required: %w", repo.ErrApprovalInvalid)
	}

	pending, err := s.repo.FindPendingByTarget(ctx, req.TargetType, req.TargetID)
	if err != nil {
		return nil, err
	}
	if len(pending) > 0 {
		return nil, ErrPendingApprovalExists
	}

	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal approval payload: %w", err)
	}

	approval := &model.ChangeApproval{
		ID:          uuid.NewString(),
		TargetType:  req.TargetType,
		TargetID:    req.TargetID,
		Payload:     datatypes.JSON(payloadBytes),
		Status:      string(repo.ApprovalStatusPending),
		SubmittedBy: req.SubmittedBy,
	}

	created, err := s.repo.Create(ctx, approval)
	if err != nil {
		return nil, err
	}

	s.log().WithFields(logrus.Fields{
		"approval_id": created.ID,
		"target_type": created.TargetType,
		"target_id":   created.TargetID,
	}).Info("integration change submitted for approval")

	return created, nil
}

// DecisionRequest 描述审批决策。
type DecisionRequest struct {
	ID       string
	Reviewer string
	Reason   string
}

// Approve 通过审批。
func (s *ApprovalService) Approve(ctx context.Context, req DecisionRequest) (*model.ChangeApproval, error) {
	if err := validateDecision(req); err != nil {
		return nil, err
	}

	approval, err := s.repo.UpdateStatus(ctx, req.ID, repo.StatusUpdate{
		Status:     repo.ApprovalStatusApproved,
		ReviewedBy: req.Reviewer,
		Reason:     req.Reason,
	})
	if err != nil {
		return nil, err
	}

	s.logDecision("approved", approval, req.Reviewer)
	s.fireHooks(ctx, approval)
	return approval, nil
}

// Reject 拒绝审批。
func (s *ApprovalService) Reject(ctx context.Context, req DecisionRequest) (*model.ChangeApproval, error) {
	if err := validateDecision(req); err != nil {
		return nil, err
	}

	approval, err := s.repo.UpdateStatus(ctx, req.ID, repo.StatusUpdate{
		Status:     repo.ApprovalStatusRejected,
		ReviewedBy: req.Reviewer,
		Reason:     req.Reason,
	})
	if err != nil {
		return nil, err
	}

	s.logDecision("rejected", approval, req.Reviewer)
	s.fireHooks(ctx, approval)
	return approval, nil
}

// ListPending 返回待审批记录。
func (s *ApprovalService) ListPending(ctx context.Context, targetType string, limit int) ([]model.ChangeApproval, error) {
	return s.repo.ListPending(ctx, targetType, limit)
}

// GetByID 获取指定审批。
func (s *ApprovalService) GetByID(ctx context.Context, id string) (*model.ChangeApproval, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ApprovalService) fireHooks(ctx context.Context, approval *model.ChangeApproval) {
	for _, hook := range s.hooks {
		func(h ApprovalHook) {
			defer func() {
				if r := recover(); r != nil && s.logger != nil {
					s.logger.WithField("panic", r).Warn("approval hook panicked")
				}
			}()
			h(ctx, approval)
		}(hook)
	}
}

func validateDecision(req DecisionRequest) error {
	if strings.TrimSpace(req.ID) == "" {
		return fmt.Errorf("%w: id is required", ErrDecisionInvalid)
	}
	if strings.TrimSpace(req.Reviewer) == "" {
		return fmt.Errorf("%w: reviewer is required", ErrDecisionInvalid)
	}
	return nil
}

func (s *ApprovalService) logDecision(action string, approval *model.ChangeApproval, reviewer string) {
	s.log().WithFields(logrus.Fields{
		"approval_id": approval.ID,
		"target_type": approval.TargetType,
		"target_id":   approval.TargetID,
		"reviewer":    reviewer,
	}).Infof("integration change %s", action)
}

func (s *ApprovalService) log() *logrus.Entry {
	if s.logger != nil {
		return s.logger
	}
	return logrus.WithField("component", "integration.approval_service")
}
