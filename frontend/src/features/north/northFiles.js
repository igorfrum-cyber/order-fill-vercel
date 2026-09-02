export function sameNorthFile(left, right) {
  return left.name === right.name && left.size === right.size && left.lastModified === right.lastModified;
}

export function uniqueNorthFiles(files) {
  const unique = [];
  for (const file of files) {
    if (!unique.some((item) => sameNorthFile(item, file))) unique.push(file);
  }
  return unique;
}
