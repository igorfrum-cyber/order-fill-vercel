

export function articleKeys(article, rule) {
  const keys = new Set([article]);
  for (const prefix of rule.articlePrefixAliases || []) {
    if (article.startsWith(prefix) && /^\d+$/.test(article.slice(prefix.length))) {
      keys.add(article.slice(prefix.length));
    } else if (/^\d+$/.test(article)) {
      keys.add(`${prefix}${article}`);
    }
  }
  return Array.from(keys).filter(Boolean);
}

export function uniqueBySourceRow(items) {
  const seen = new Set();
  return items.filter((item) => {
    if (seen.has(item.rowIndex)) return false;
    seen.add(item.rowIndex);
    return true;
  });
}

export function buildSourceContext(source, rule) {
  const sourceIndex = new Map();
  const noArticleItems = [];
  for (const item of source.items) {
    if (item.article) {
      for (const key of articleKeys(item.article, rule)) {
        if (!sourceIndex.has(key)) sourceIndex.set(key, []);
        sourceIndex.get(key).push(item);
      }
    } else {
      noArticleItems.push(item);
    }
  }
  return {
    sourceIndex,
    noArticleItems,
    sourceDuplicateGroups: sourceDuplicateGroups(sourceIndex),
    sourceArticleCount: new Set(source.items.map((item) => item.article).filter(Boolean)).size,
  };
}

export function sourceDuplicateGroups(sourceIndex) {
  const seenGroups = new Set();
  const groups = [];
  for (const [article, items] of sourceIndex.entries()) {
    const uniqueItems = uniqueBySourceRow(items);
    if (uniqueItems.length < 2) continue;
    const signature = uniqueItems.map((item) => item.rowIndex).sort((left, right) => left - right).join(":");
    if (seenGroups.has(signature)) continue;
    seenGroups.add(signature);
    groups.push({
      article,
      candidates: duplicateCandidatesForReport(uniqueItems),
    });
  }
  return groups;
}

export function candidatesForArticle(sourceContext, article, rule) {
  return uniqueBySourceRow(articleKeys(article, rule).flatMap((key) => sourceContext.sourceIndex.get(key) || []));
}

export function duplicateCandidatesForReport(candidates) {
  return candidates.map((item) => ({
    sourceRow: item.rowIndex,
    sourceArticle: item.articleRaw,
    sourceName: item.name,
    recommended: item.recommended,
    rounded: item.rounded,
    stock: item.stock,
    inTransit: item.inTransit,
  }));
}
