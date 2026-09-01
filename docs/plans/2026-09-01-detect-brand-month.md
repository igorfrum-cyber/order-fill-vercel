# Auto-detect brand and order month

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Order-fill starts from files only: the worker reads brand and order month from the 1C sales table, and Christina still takes separate HOME and PROFF blanks.

**Architecture:** Detection lives in document-service (`orderfill` + `brand`). api-service accepts empty `brand` / `order_month` for `order_fill`. The browser never parses Excel; it only shows HOME/PROFF slots when the source filename looks like Christina.

**Tech Stack:** Go domain tests, OpenAPI, Vite frontend.

---
