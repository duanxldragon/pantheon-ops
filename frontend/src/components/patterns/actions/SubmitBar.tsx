// SubmitBar - Shell visual contract compliance
import React from 'react';

export interface SubmitBarProps {
  children?: React.ReactNode;
}

export const SubmitBar: React.FC<SubmitBarProps> = ({ children }) => {
  return <div className="submit-bar">{children}</div>;
};

export {};
