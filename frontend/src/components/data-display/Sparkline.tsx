import React, { useId } from 'react';

interface SparklineProps {
  /** Series values, oldest → newest. */
  data: number[];
  width?: number;
  height?: number;
  /** Stroke color; defaults to the brand token so it follows the theme. */
  color?: string;
  /** Fill the area under the line with a faint gradient. */
  area?: boolean;
  strokeWidth?: number;
  className?: string;
  ariaLabel?: string;
}

/**
 * Dependency-free SVG sparkline. A lightweight trend primitive for dashboard
 * cards — no charting library, follows the theme via CSS tokens by default.
 * Renders nothing meaningful for <2 points.
 */
const Sparkline: React.FC<SparklineProps> = ({
  data,
  width = 96,
  height = 28,
  color = 'var(--brand-primary)',
  area = true,
  strokeWidth = 1.5,
  className,
  ariaLabel,
}) => {
  const gradientId = useId();
  if (!data || data.length < 2) {
    return null;
  }

  const min = Math.min(...data);
  const max = Math.max(...data);
  const span = max - min || 1;
  const stepX = width / (data.length - 1);
  const pad = strokeWidth;
  const usableH = height - pad * 2;

  const points = data.map((value, index) => {
    const x = index * stepX;
    const y = pad + usableH - ((value - min) / span) * usableH;
    return [x, y];
  });

  const line = points
    .map(([x, y], i) => `${i === 0 ? 'M' : 'L'}${x.toFixed(2)},${y.toFixed(2)}`)
    .join(' ');
  const areaPath = area ? `${line} L${width},${height} L0,${height} Z` : undefined;

  return (
    <svg
      className={className}
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      role="img"
      aria-label={ariaLabel ?? 'trend'}
      preserveAspectRatio="none"
    >
      {area ? (
        <>
          <defs>
            <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity="0.24" />
              <stop offset="100%" stopColor={color} stopOpacity="0" />
            </linearGradient>
          </defs>
          <path d={areaPath} fill={`url(#${gradientId})`} stroke="none" />
        </>
      ) : null}
      <path
        d={line}
        fill="none"
        stroke={color}
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
};

export default Sparkline;
