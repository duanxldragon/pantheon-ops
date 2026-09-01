---
title: Upload and Storage Design
doc_type: Design
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-04-29
---

# Upload and Storage Design

Chinese version: [UPLOAD_AND_STORAGE_DESIGN.md](./UPLOAD_AND_STORAGE_DESIGN.md)

This document defines the boundary between `system/config -> upload` and the platform’s unified upload capability.

It answers:

- why upload belongs under `system/config`
- who owns configuration, runtime file handling, and public object URLs
- how local storage and `s3-compatible` storage switch
- what acceptance must verify

## 1. Module Positioning

Upload is a shared platform foundation capability.

Ownership split:

- configuration belongs to `system/config`
- runtime file handling belongs to platform common packages
- business modules reuse one unified entry instead of inventing their own upload protocols

The model is:

- one upload entry
- one shared configuration model
- one shared set of file-size, file-type, path, and URL-generation rules

## 2. Boundary

### 2.1 What Upload Owns

- upload configuration reads
- file-size limits
- file-type allowlists
- local write-path resolution
- local file-access URL generation
- `s3-compatible` object upload
- the unified upload API

### 2.2 What Upload Does Not Own

- business-specific file metadata meaning
- per-module attachment lifecycle rules
- business approval or archival semantics

### 2.3 Collaboration Boundary

- `system/config -> setting` owns the configuration values
- `backend/pkg/upload` owns runtime handling
- the frontend uploads through the unified interface
- business modules declare `scope` and consume the returned result

## 3. Current Runtime Capability

Unified upload entry:

- `POST /api/v1/system/upload`

Local file-read entry:

- `GET /api/v1/system/upload/files/*filepath`

Current frontend integrations include profile-avatar and user-avatar upload flows through shared upload wrappers.

## 4. Configuration Model

Upload configuration lives in `system_setting` under the `upload` group.

Core settings:

- `upload.storage_driver`
- `upload.max_file_size`
- `upload.allowed_types`
- `upload.local_path`
- `upload.public_base_url`
- `upload.s3_endpoint`
- `upload.s3_bucket`
- `upload.s3_region`
- `upload.s3_access_key_id`
- `upload.s3_secret_access_key`

### 4.1 Current Drivers

Supported:

- `local`
- `s3`

Compatibility note:

- `s3-compatible` may normalize into `s3`

### 4.2 Current Default Semantics

- `local`: write files locally, then expose them through platform routes
- `s3`: write to object storage, then return either object URLs or assembled public URLs

## 5. Local Storage Design

### 5.1 Path Constraints

Local mode must stay constrained by:

- `upload.local_path`

Requirements:

- block directory traversal
- never resolve outside the configured root
- keep object keys and physical paths traceable

### 5.2 URL Generation

Local mode exposes files through:

- `/api/v1/system/upload/files/*filepath`

If `upload.public_base_url` is configured, it may be used to assemble the final public URL.

## 6. S3-Compatible Storage Design

### 6.1 Required Configuration

- endpoint
- bucket
- region
- access key id
- secret access key

### 6.2 Runtime Requirements

- initialize the bucket target
- write objects successfully
- generate usable object-access URLs

### 6.3 Error Semantics

At minimum, distinguish:

- endpoint missing
- bucket missing
- credentials missing
- invalid endpoint
- bucket initialization failure
- upload failure

## 7. File Validation Rules

The upload entry must centrally validate:

1. file presence
2. file-size limit
3. allowed extension or type
4. target-path safety

Business modules must not reimplement these checks inconsistently.

## 8. `scope` Design

The upload API uses `scope` to distinguish usage contexts, for example:

- `general`
- `profile/avatar`

Rules:

- `scope` identifies scenario, not permission boundary
- do not treat `scope` itself as the security model
- real security still depends on login state, route permissions, and upload configuration validation

## 9. Security and Audit

### 9.1 Security Requirements

- upload requires at least an authenticated session
- file size and type cannot be trusted to frontend-only control
- local paths must block traversal attacks
- sensitive object-storage credentials must be encrypted at rest

### 9.2 Audit Requirements

Upload actions must enter unified operation audit.

At minimum record:

- operator
- `scope`
- filename
- file size
- driver type
- result status

Goal:

- be able to answer who uploaded what, where it was written, and whether it succeeded

## 10. Frontend Consumption Constraints

The frontend must not:

- assemble local file paths
- decide allowed file size by itself
- decide allowed file extensions by itself

The frontend should:

- upload through the unified API
- consume returned object URLs or object keys
- keep failure feedback inside the translated `message`-key flow

## 11. Acceptance Requirements

### 11.1 Base Flow

Verify:

- upload-group settings can be read and written
- upload entry works end to end
- local file access works where applicable
- the shared default `upload.allowed_types` covers `jpg/jpeg/png/webp/gif/pdf/doc/docx/xls/xlsx/zip/gz/tgz/tar`, with `webp` and `gif` keeping the base allowlist aligned with shared frontend upload consumers so downstream repos do not need local overrides

### 11.2 Validation Flow

Verify file-size limits, type validation, and path-safety failures.

### 11.3 Consumption Flow

Verify avatar and shared upload consumers can use the returned results correctly.

### 11.4 Audit Flow

Verify upload actions land in operation audit with sufficient traceability.

## 12. Current Conclusion

Upload is a shared platform capability governed through `system/config`, with runtime handling in common packages and business modules constrained to one unified, auditable upload path.
