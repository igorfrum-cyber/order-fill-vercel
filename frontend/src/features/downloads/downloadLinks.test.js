import test from "node:test";
import assert from "node:assert/strict";

import { remoteDownloadLinksHtml } from "./downloadLinks.js";

test("remoteDownloadLinksHtml renders escaped server download links", () => {
  const html = remoteDownloadLinksHtml([
    {
      download_url: "https://files.example/out.xlsx",
      name: "order <final>.xlsx",
      label: "Скачать заказ",
    },
  ]);

  assert.equal(html, '<a class="file-link" href="https://files.example/out.xlsx" download="order &lt;final&gt;.xlsx">Скачать заказ</a>');
});

test("remoteDownloadLinksHtml supports camelCase downloadUrl fallback", () => {
  assert.match(remoteDownloadLinksHtml([{ downloadUrl: "/file", name: "file.xlsx" }]), /href="\/file"/);
});
