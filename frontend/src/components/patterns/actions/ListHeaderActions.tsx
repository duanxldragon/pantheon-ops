import React from 'react';
import PageActions from './PageActions';

interface ListHeaderActionsProps {
  utility?: React.ReactNode;
  primary?: React.ReactNode;
  className?: string;
}

const ListHeaderActions: React.FC<ListHeaderActionsProps> = ({ utility, primary, className }) => (
  <div className={className ? `list-header-actions ${className}` : 'list-header-actions'}>
    <PageActions className="list-header-actions__utility">{utility}</PageActions>
    {primary ? <PageActions className="list-header-actions__primary">{primary}</PageActions> : null}
  </div>
);

export default ListHeaderActions;
