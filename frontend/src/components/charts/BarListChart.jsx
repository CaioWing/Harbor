const COLORS = ["#0f766e", "#2563eb", "#f59e0b", "#7c3aed", "#ef4444", "#475569"];

export default function BarListChart({ title, data = [] }) {
  const max = Math.max(1, ...data.map((item) => Number(item.value || 0)));

  return (
    <article className="panel chart-card">
      <h3>{title}</h3>
      <ul className="bar-list">
        {data.map((item, index) => {
          const value = Number(item.value || 0);
          const width = (value / max) * 100;

          return (
            <li key={item.label}>
              <div className="bar-list__meta">
                <span>{item.label}</span>
                <strong>{value}</strong>
              </div>
              <div className="bar-list__track">
                <span
                  className="bar-list__fill"
                  style={{
                    width: `${width}%`,
                    backgroundColor: COLORS[index % COLORS.length]
                  }}
                />
              </div>
            </li>
          );
        })}
      </ul>
    </article>
  );
}
