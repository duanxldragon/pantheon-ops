package deploy

import (
	"context"
	"errors"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

type credentialScopeOption struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

func (s *DeployService) listCredentialScopeOptions(ctx context.Context, scope *common.DataScopeReq) ([]credentialScopeOption, error) {
	reader, ok := s.bizScopeReader.(bizcap.ActiveBizScopeLister)
	if !ok {
		return nil, errors.New("business.deploy.credential.business_scope_required")
	}
	owners, err := reader.ListActive(ctx, scope)
	if err != nil {
		return nil, err
	}
	items := make([]credentialScopeOption, 0, len(owners))
	for _, owner := range owners {
		items = append(items, credentialScopeOption{ID: owner.ID, Name: owner.Name})
	}
	return items, nil
}
