import test from "node:test";
import assert from "node:assert/strict";

import { remoteDownloadLinksHtml, fileNameFromContentDisposition } from "./downloadLinks.js";

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

test("fileNameFromContentDisposition prefers the RFC 5987 file name", () => {
  assert.equal(
    fileNameFromContentDisposition(`attachment; filename="angio.zip"; filename*=UTF-8''angiopharm_2026-09.zip`),
    "angiopharm_2026-09.zip",
  );
});

test("fileNameFromContentDisposition falls back to the undecoded value when escaping is invalid", () => {
  assert.equal(fileNameFromContentDisposition("attachment; filename*=UTF-8''bad%ZZ"), "bad%ZZ");
  assert.equal(fileNameFromContentDisposition('attachment; filename="quoted.zip"'), "quoted.zip");
  assert.equal(fileNameFromContentDisposition("attachment; filename= plain.zip "), "plain.zip");
  assert.equal(fileNameFromContentDisposition(null), "");
});
