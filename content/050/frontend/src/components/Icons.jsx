import React from 'react';

const BASE = { width: 14, height: 14, viewBox: '0 0 24 24' };

export function Smiley({ filled = true }) {
  const fill = filled ? '#E6117F' : '#DADCE5';
  const stroke = filled ? '#9A0157' : '#B4B7C4';
  return (
    <svg {...BASE} aria-hidden="true">
      <circle cx="12" cy="12" r="10" fill={fill} stroke={stroke} strokeWidth="1" />
      <circle cx="9" cy="10" r="1.4" fill="#fff" />
      <circle cx="15" cy="10" r="1.4" fill="#fff" />
      <path d="M8 14 Q12 18 16 14" stroke="#fff" strokeWidth="1.6" fill="none" strokeLinecap="round" />
    </svg>
  );
}

export function Snowflake({ filled = true }) {
  const stroke = filled ? '#1E73C0' : '#B4B7C4';
  return (
    <svg {...BASE} aria-hidden="true">
      <g stroke={stroke} strokeWidth="1.6" strokeLinecap="round" fill="none">
        <line x1="12" y1="3" x2="12" y2="21" />
        <line x1="3" y1="12" x2="21" y2="12" />
        <line x1="5.5" y1="5.5" x2="18.5" y2="18.5" />
        <line x1="18.5" y1="5.5" x2="5.5" y2="18.5" />
        <polyline points="10,5 12,3 14,5" />
        <polyline points="10,19 12,21 14,19" />
        <polyline points="5,10 3,12 5,14" />
        <polyline points="19,10 21,12 19,14" />
      </g>
    </svg>
  );
}

export function Heart({ filled = true }) {
  const fill = filled ? '#E6117F' : '#DADCE5';
  const stroke = filled ? '#9A0157' : '#B4B7C4';
  return (
    <svg {...BASE} aria-hidden="true">
      <path
        d="M12 21s-7-4.35-9.5-9.2C1 8 3 4 7 4c2 0 3.5 1.2 5 3 1.5-1.8 3-3 5-3 4 0 6 4 4.5 7.8C19 16.65 12 21 12 21z"
        fill={fill}
        stroke={stroke}
        strokeWidth="1"
      />
    </svg>
  );
}

export function Star({ filled = true }) {
  const fill = filled ? '#F2B400' : '#DADCE5';
  const stroke = filled ? '#B7860B' : '#B4B7C4';
  return (
    <svg {...BASE} aria-hidden="true">
      <polygon
        points="12,2 15.1,8.6 22,9.3 16.8,14 18.3,21 12,17.3 5.7,21 7.2,14 2,9.3 8.9,8.6"
        fill={fill}
        stroke={stroke}
        strokeWidth="1"
      />
    </svg>
  );
}

export function AddFriend() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="9" cy="8" r="4" fill="#E6117F" />
      <path d="M1 22c1-6 6-9 8-9s7 3 8 9z" fill="#E6117F" />
      <path d="M18 7v8M14 11h8" stroke="#9A0157" strokeWidth="2" strokeLinecap="round" />
    </svg>
  );
}
