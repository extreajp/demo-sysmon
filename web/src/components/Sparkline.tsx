import styles from "./Sparkline.module.css";

type Props = { values: number[]; color?: string; suffix?: string };

function fmt(n: number, suffix: string): string {
  const abs = Math.abs(n);
  const body = abs >= 100 ? n.toFixed(0) : abs >= 10 ? n.toFixed(1) : n.toFixed(2);
  return `${body}${suffix}`;
}

export function Sparkline({ values, color = "#6ee7b7", suffix = "" }: Props) {
  const w = 220;
  const h = 48;
  const min = values.length ? Math.min(...values) : 0;
  const max = values.length ? Math.max(...values) : 0;
  const span = max - min || 1;
  const pts = values
    .map((v, i) => {
      const x = (i / Math.max(1, values.length - 1)) * (w - 4) + 2;
      const y = h - 4 - ((v - min) / span) * (h - 8);
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
  return (
    <div className={styles.row}>
      <svg width={w} height={h} viewBox={`0 0 ${w} ${h}`}>
        {values.length > 0 && (
          <polyline fill="none" stroke={color} strokeWidth="1.6" points={pts} />
        )}
      </svg>
      <div className={styles.axis}>
        <span>{fmt(max, suffix)}</span>
        <span>{fmt(min, suffix)}</span>
      </div>
    </div>
  );
}
