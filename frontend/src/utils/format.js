export function formatDateTime(value) {
  if (!value) {
    return "-";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

export function shortenID(value, size = 8) {
  if (!value || typeof value !== "string") {
    return "-";
  }

  if (value.length <= size) {
    return value;
  }

  return `${value.slice(0, size)}...`;
}
