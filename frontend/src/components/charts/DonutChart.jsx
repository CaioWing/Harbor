const COLORS = ["#0f766e", "#d97706", "#dc2626", "#334155", "#7c3aed", "#4d7c0f"];

export default function DonutChart({ title, data = [] }) {
  const total = data.reduce((sum, item) => sum + Number(item.value || 0), 0);

  let cursor = 0;
  const segments = data
    .filter((item) => Number(item.value || 0) > 0)
    .map((item, index) => {
      const value = Number(item.value || 0);
      const ratio = total > 0 ? value / total : 0;
      const dash = `${ratio * 100} ${100 - ratio * 100}`;
      const itemCursor = cursor;
      cursor += ratio * 100;
      return {
        label: item.label,
        value,
        color: COLORS[index % COLORS.length],
        dash,
        offset: -itemCursor
      };
    });

  return (
    <article className="panel chart-card">
      <h3>{title}</h3>
      <div className="donut">
        <svg viewBox="0 0 42 42" className="donut__svg" aria-label={title}>
          <circle className="donut__base" cx="21" cy="21" r="15.915" />
          {segments.map((segment) => (
            <circle
              key={segment.label}
              cx="21"
              cy="21"
              r="15.915"
              fill="transparent"
              stroke={segment.color}
              strokeWidth="4"
              strokeDasharray={segment.dash}
              strokeDashoffset={segment.offset}
              transform="rotate(-90 21 21)"
            />
          ))}
        </svg>

        <div className="donut__center">
          <span className="donut__total">{total}</span>
          <span className="donut__caption">total</span>
        </div>
      </div>

      <ul className="legend">
        {segments.map((segment) => (
          <li key={segment.label}>
            <span className="legend__swatch" style={{ backgroundColor: segment.color }} />
            <span className="legend__label">{segment.label}</span>
            <span className="legend__value">{segment.value}</span>
          </li>
        ))}
      </ul>
    </article>
  );
}
