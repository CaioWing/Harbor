const POSITIVE = ["success", "completed", "accepted", "active"];
const NEGATIVE = ["failure", "rejected", "cancelled"];
const WARNING = ["pending", "scheduled", "downloading", "installing"];

export default function StatusPill({ status }) {
  const value = String(status || "unknown").toLowerCase();

  let tone = "pill";
  if (POSITIVE.includes(value)) {
    tone = "pill pill--positive";
  } else if (NEGATIVE.includes(value)) {
    tone = "pill pill--negative";
  } else if (WARNING.includes(value)) {
    tone = "pill pill--warning";
  }

  return <span className={tone}>{value}</span>;
}
