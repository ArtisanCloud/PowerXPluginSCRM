package social_channel_governance

import (
	"context"
	"strings"
	"time"

	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
)

type TagRecord struct {
	TagID   string `json:"tag_id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type TagSyncResult struct {
	Pulled          int    `json:"pulled"`
	Pushed          int    `json:"pushed"`
	Conflicts       int    `json:"conflicts"`
	SnapshotVersion string `json:"snapshot_version"`
	Cursor          string `json:"cursor"`
}

type TagSyncService struct {
	mappingRepo *socialrepo.TagMappingRepository
	conflictSvc *ConflictResolutionService
}

func NewTagSyncService(mappingRepo *socialrepo.TagMappingRepository, conflictSvc *ConflictResolutionService) *TagSyncService {
	return &TagSyncService{mappingRepo: mappingRepo, conflictSvc: conflictSvc}
}

func (s *TagSyncService) SyncRemoteToLocal(
	ctx context.Context,
	tenantUUID string,
	remoteTags []TagRecord,
	localTags []TagRecord,
	cursor string,
) (*TagSyncResult, error) {
	res := &TagSyncResult{Cursor: strings.TrimSpace(cursor)}
	localByID := toTagMap(localTags)
	for _, remote := range remoteTags {
		rid := strings.TrimSpace(remote.TagID)
		if rid == "" {
			continue
		}
		local, ok := localByID[rid]
		if ok && strings.TrimSpace(local.Name) != strings.TrimSpace(remote.Name) {
			res.Conflicts++
			if s.conflictSvc != nil {
				_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag", rid,
					map[string]any{"name": local.Name, "version": local.Version},
					map[string]any{"name": remote.Name, "version": remote.Version},
				)
			}
			continue
		}
		res.Pulled++
	}
	if s.mappingRepo != nil {
		version, err := s.mappingRepo.NextSnapshotVersion(ctx, tenantUUID, "pull")
		if err != nil {
			return nil, err
		}
		res.SnapshotVersion = version
		if err := s.mappingRepo.SaveCheckpoint(ctx, tenantUUID, "pull", res.Cursor, version, time.Now().UTC()); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (s *TagSyncService) SyncLocalToRemote(
	ctx context.Context,
	tenantUUID string,
	localTags []TagRecord,
	remoteTags []TagRecord,
	cursor string,
) (*TagSyncResult, error) {
	res := &TagSyncResult{Cursor: strings.TrimSpace(cursor)}
	remoteByID := toTagMap(remoteTags)
	for _, local := range localTags {
		lid := strings.TrimSpace(local.TagID)
		if lid == "" {
			continue
		}
		remote, ok := remoteByID[lid]
		if ok && strings.TrimSpace(remote.Name) != strings.TrimSpace(local.Name) {
			res.Conflicts++
			if s.conflictSvc != nil {
				_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag", lid,
					map[string]any{"name": local.Name, "version": local.Version},
					map[string]any{"name": remote.Name, "version": remote.Version},
				)
			}
			continue
		}
		res.Pushed++
	}
	if s.mappingRepo != nil {
		version, err := s.mappingRepo.NextSnapshotVersion(ctx, tenantUUID, "push")
		if err != nil {
			return nil, err
		}
		res.SnapshotVersion = version
		if err := s.mappingRepo.SaveCheckpoint(ctx, tenantUUID, "push", res.Cursor, version, time.Now().UTC()); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func toTagMap(items []TagRecord) map[string]TagRecord {
	out := make(map[string]TagRecord, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.TagID)
		if key == "" {
			continue
		}
		out[key] = item
	}
	return out
}
