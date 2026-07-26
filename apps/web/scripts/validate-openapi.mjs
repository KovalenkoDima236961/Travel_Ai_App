#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { parseDocument } from "yaml";

const METHODS = new Set(["get", "post", "put", "patch", "delete"]);

function isObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function hasOwn(object, key) {
  return Object.prototype.hasOwnProperty.call(object, key);
}

function isPresent(value) {
  if (Array.isArray(value)) {
    return value.length > 0;
  }
  if (typeof value === "string") {
    return value.trim().length > 0;
  }
  return Boolean(value);
}

function formatYamlError(error) {
  const location = error.linePos?.[0];
  if (!location) {
    return error.message;
  }
  return `${location.line}:${location.col} ${error.message}`;
}

function loadSpec(filePath) {
  const contents = fs.readFileSync(filePath, "utf8");
  const document = parseDocument(contents, { prettyErrors: false });

  if (document.errors.length > 0) {
    const errors = document.errors.map(formatYamlError).join("; ");
    throw new Error(errors);
  }

  return document.toJSON();
}

function validateOperation({ filePath, issues, method, operation, route }) {
  if (!isPresent(operation.operationId)) {
    issues.push(`${filePath}: ${method.toUpperCase()} ${route} is missing operationId`);
  }

  if (!Array.isArray(operation.tags) || operation.tags.length === 0) {
    issues.push(`${filePath}: ${method.toUpperCase()} ${route} is missing tags`);
  }

  if (!isObject(operation.responses) || !hasOwn(operation.responses, "default") || !isPresent(operation.responses.default)) {
    issues.push(`${filePath}: ${method.toUpperCase()} ${route} is missing responses.default`);
  }
}

function validateSpec(filePath, spec) {
  const issues = [];

  if (!isObject(spec)) {
    return [`${filePath}: document root must be an object`];
  }

  if (!isObject(spec.paths)) {
    return [`${filePath}: document is missing paths`];
  }

  for (const [route, pathItem] of Object.entries(spec.paths)) {
    if (!isObject(pathItem)) {
      continue;
    }

    for (const [method, operation] of Object.entries(pathItem)) {
      const normalizedMethod = method.toLowerCase();
      if (!METHODS.has(normalizedMethod)) {
        continue;
      }
      if (!isObject(operation)) {
        issues.push(`${filePath}: ${normalizedMethod.toUpperCase()} ${route} operation must be an object`);
        continue;
      }
      validateOperation({
        filePath,
        issues,
        method: normalizedMethod,
        operation,
        route
      });
    }
  }

  return issues;
}

const specFiles = process.argv.slice(2);

if (specFiles.length === 0) {
  console.error("No OpenAPI specifications were provided.");
  process.exit(2);
}

const allIssues = [];

for (const specFile of specFiles) {
  const displayPath = path.relative(process.cwd(), specFile);
  try {
    const spec = loadSpec(specFile);
    allIssues.push(...validateSpec(displayPath, spec));
  } catch (error) {
    allIssues.push(`${displayPath}: ${error.message}`);
  }
}

if (allIssues.length > 0) {
  console.error("OpenAPI validation failed:");
  for (const issue of allIssues) {
    console.error(`- ${issue}`);
  }
  process.exit(1);
}

console.log(`Validated ${specFiles.length} OpenAPI specification${specFiles.length === 1 ? "" : "s"}.`);
