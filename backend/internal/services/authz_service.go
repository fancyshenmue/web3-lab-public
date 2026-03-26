package services

import (
	"context"
	"fmt"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AuthzService wraps the SpiceDB gRPC client.
type AuthzService struct {
	client *authzed.Client
}

func NewAuthzService(endpoint, token string, insecureConn bool) (*AuthzService, error) {
	opts := []grpc.DialOption{
		grpcutil.WithInsecureBearerToken(token),
	}
	if insecureConn {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	client, err := authzed.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("create spicedb client: %w", err)
	}

	return &AuthzService{client: client}, nil
}

// CheckPermission checks if a subject has a specific permission on a resource.
func (s *AuthzService) CheckPermission(ctx context.Context, resourceType, resourceID, permission, subjectType, subjectID string) (bool, error) {
	resp, err := s.client.CheckPermission(ctx, &v1.CheckPermissionRequest{
		Resource:   &v1.ObjectReference{ObjectType: resourceType, ObjectId: resourceID},
		Permission: permission,
		Subject:    &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: subjectType, ObjectId: subjectID}},
	})
	if err != nil {
		return false, fmt.Errorf("check permission: %w", err)
	}
	return resp.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION, nil
}

// CreateRelationship writes a relationship tuple to SpiceDB.
func (s *AuthzService) CreateRelationship(ctx context.Context, resourceType, resourceID, relation, subjectType, subjectID string) error {
	_, err := s.client.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
		Updates: []*v1.RelationshipUpdate{
			{
				Operation: v1.RelationshipUpdate_OPERATION_TOUCH,
				Relationship: &v1.Relationship{
					Resource: &v1.ObjectReference{ObjectType: resourceType, ObjectId: resourceID},
					Relation: relation,
					Subject:  &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: subjectType, ObjectId: subjectID}},
				},
			},
		},
	})
	return err
}

// DeleteRelationship removes a relationship tuple from SpiceDB.
func (s *AuthzService) DeleteRelationship(ctx context.Context, resourceType, resourceID, relation, subjectType, subjectID string) error {
	_, err := s.client.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
		Updates: []*v1.RelationshipUpdate{
			{
				Operation: v1.RelationshipUpdate_OPERATION_DELETE,
				Relationship: &v1.Relationship{
					Resource: &v1.ObjectReference{ObjectType: resourceType, ObjectId: resourceID},
					Relation: relation,
					Subject:  &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: subjectType, ObjectId: subjectID}},
				},
			},
		},
	})
	return err
}

// Close closes the SpiceDB gRPC connection.
func (s *AuthzService) Close() error {
	// authzed-go client doesn't expose Close directly — connection managed by gRPC
	return nil
}
