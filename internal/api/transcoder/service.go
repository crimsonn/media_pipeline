package transcoder

import (
	"context"
	"errors"
	"fmt"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/domain"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
)

type Service interface {
	GetFullProfile(ctx context.Context, profileID int64) (*domain.Profile, error)
	CreateRendition(ctx context.Context, req domain.CreateRenditionRequest) (domain.RenditionResponse, error)
	GetRenditions(ctx context.Context) ([]domain.RenditionResponse, error)
	DeleteRendition(ctx context.Context, id int64) error
	CreateProfile(ctx context.Context, req domain.CreateProfileRequest) (domain.ProfileResponse, error)
	GetProfiles(ctx context.Context) ([]domain.ProfileResponse, error)
}

type service struct {
	queries *queries.Queries
}

func (s *service) GetProfiles(ctx context.Context) ([]domain.ProfileResponse, error) {
	rows, err := s.queries.GetAllTranscodeProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("get profiles: %w", err)
	}
	byID := make(map[int64]*domain.ProfileResponse)
	order := make([]int64, 0)

	for _, row := range rows {
		profile, ok := byID[row.ProfileID]
		if !ok {
			description := ""
			if row.ProfileDescription != nil {
				description = *row.ProfileDescription
			}
			profile = &domain.ProfileResponse{
				ID:             row.ProfileID,
				Name:           row.ProfileName,
				Description:    description,
				HlsSegmentTime: int(row.HlsSegmentTime),
				Renditions:     make([]domain.RenditionResponse, 0),
			}
			byID[row.ProfileID] = profile
			order = append(order, row.ProfileID)
		}
		profile.Renditions = append(profile.Renditions, domain.RenditionResponse{
			ID:           row.RenditionID,
			Name:         row.RenditionName,
			Width:        int(row.Width),
			Height:       int(row.Height),
			VideoBitrate: int(row.VideoBitrate),
			AudioBitrate: int(row.AudioBitrate),
			VideoCodec:   row.VideoCodec,
			AudioCodec:   row.AudioCodec,
			Fps:          int(row.Fps),
		})
	}

	responses := make([]domain.ProfileResponse, 0, len(order))
	for _, id := range order {
		responses = append(responses, *byID[id])
	}
	return responses, nil
}

func (s *service) GetFullProfile(ctx context.Context, profileID int64) (*domain.Profile, error) {
	rows, err := s.queries.GetProfileWithRenditions(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrProfileNotFound
	}

	first := rows[0]
	description := ""
	if first.ProfileDescription != nil {
		description = *first.ProfileDescription
	}

	profile := &domain.Profile{
		ID:             first.ProfileID,
		Name:           first.ProfileName,
		Description:    description,
		HlsSegmentTime: int(first.HlsSegmentTime),
		Renditions:     make([]domain.Rendition, 0, len(rows)),
	}

	for _, row := range rows {
		profile.Renditions = append(profile.Renditions, domain.Rendition{
			ID:           row.RenditionID,
			Name:         row.RenditionName,
			Width:        int(row.Width),
			Height:       int(row.Height),
			VideoBitrate: int(row.VideoBitrate),
			AudioBitrate: int(row.AudioBitrate),
			VideoCodec:   row.VideoCodec,
			AudioCodec:   row.AudioCodec,
			StreamIndex:  int(row.StreamIndex),
		})
	}

	return profile, nil
}

func (s *service) CreateRendition(ctx context.Context, req domain.CreateRenditionRequest) (domain.RenditionResponse, error) {
	rendition, err := s.queries.CreateRendition(ctx, queries.CreateRenditionParams{
		Name:         req.Name,
		Width:        int32(req.Width),
		Height:       int32(req.Height),
		VideoBitrate: int32(req.VideoBitrate),
		AudioBitrate: int32(req.AudioBitrate),
		VideoCodec:   req.VideoCodec,
		AudioCodec:   req.AudioCodec,
		Fps:          int32(req.Fps),
	})
	if err != nil {
		return domain.RenditionResponse{}, fmt.Errorf("create rendition: %w", err)
	}
	return domain.ToRenditionResponse(rendition), nil
}

func (s *service) GetRenditions(ctx context.Context) ([]domain.RenditionResponse, error) {
	renditions, err := s.queries.GetAllRenditions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get renditions: %w", err)
	}
	responses := make([]domain.RenditionResponse, 0, len(renditions))
	for _, rendition := range renditions {
		responses = append(responses, domain.ToRenditionResponse(rendition))
	}
	return responses, nil
}

func (s *service) DeleteRendition(ctx context.Context, id int64) error {
	err := s.queries.DeleteRenditionById(ctx, id)
	if err != nil {
		return fmt.Errorf("delete rendition: %w", err)
	}
	return nil
}

func (s *service) CreateProfile(ctx context.Context, req domain.CreateProfileRequest) (domain.ProfileResponse, error) {
	profile, err := s.queries.CreateTranscodeProfile(ctx, queries.CreateTranscodeProfileParams{
		Name:           req.Name,
		Description:    &req.Description,
		HlsSegmentTime: int32(req.HlsSegmentTime),
		IsDefault:      req.IsDefault,
	})
	if err != nil {
		return domain.ProfileResponse{}, fmt.Errorf("create profile: %w", err)
	}
	for i, renditionID := range req.Renditions {
		err := s.queries.LinkProfileRendition(ctx, queries.LinkProfileRenditionParams{
			ProfileID:   profile.ID,
			RenditionID: int64(renditionID),
			StreamIndex: int32(i),
		})
		if err != nil {
			return domain.ProfileResponse{}, fmt.Errorf("link profile rendition: %w", err)
		}
	}

	rows, err := s.queries.GetProfileWithRenditions(ctx, profile.ID)
	if err != nil {
		return domain.ProfileResponse{}, fmt.Errorf("get renditions by profile id: %w", err)
	}

	renditionResponses := make([]queries.Rendition, 0, len(rows))
	for _, row := range rows {
		renditionResponses = append(renditionResponses, queries.Rendition{
			ID:           row.RenditionID,
			Name:         row.RenditionName,
			Width:        row.Width,
			Height:       row.Height,
			VideoBitrate: row.VideoBitrate,
			AudioBitrate: row.AudioBitrate,
			VideoCodec:   row.VideoCodec,
			AudioCodec:   row.AudioCodec,
			Fps:          row.Fps,
		})
	}
	return domain.ToProfileResponse(profile, renditionResponses), nil
}

func NewService(queries *queries.Queries) Service {
	return &service{queries}
}
