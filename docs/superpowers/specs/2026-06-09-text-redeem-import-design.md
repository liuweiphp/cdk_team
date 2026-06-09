# Text Redeem Import Design

## Goal

The redeem item import panel must support two input modes: direct text input and file upload. Text input is the default mode. Both modes require the user to choose a category and an active template before generating redeem items and CDKs.

## Current State

The admin redeem item page currently imports content only through a single file picker. The frontend sends `category_id`, `template_id`, and `file` to `POST /api/admin/redeem-items/import`.

The backend handler validates category and template IDs, requires a file, and passes the uploaded file to `RedeemItemService.ImportLines`. The service reads non-empty lines, applies the selected template to each line, creates one redeem item per line, and creates one unused CDK per redeem item.

## Approach

Keep the existing `/api/admin/redeem-items/import` endpoint and extend it to accept either:

- `text`: multiline text entered directly in the browser.
- `file`: the existing uploaded `.txt` or `.csv` file.

The endpoint will use the text source when `text` is non-empty. If `text` is empty, it will fall back to the uploaded file. This keeps one import result shape and one validation flow for both modes.

## Backend Design

`RedeemItemService` will split the current file-specific logic into a reader-based import path:

- `ImportLines(header, templateID, categoryID, createdBy)` remains as the file entry point.
- A new text entry point, such as `ImportText(text, templateID, categoryID, createdBy)`, validates text size and calls the same shared reader logic.
- A shared method imports from an `io.Reader` and receives a `sourceName` used for import records and generated item names.

Validation rules:

- Missing template returns `请选择模板`.
- Missing category returns `请选择分类`.
- Empty text in text mode returns `请输入文本内容`.
- Missing file in file mode returns `请上传文件`.
- File and text inputs both keep the existing 5MB size limit.
- Blank lines are ignored.
- If text is empty or contains only blank lines, text mode returns `请输入文本内容`.
- If an uploaded file contains no valid content lines, file mode keeps `文件没有有效内容行`.

## Frontend Design

The upload panel becomes an import panel:

- Title changes from `上传内容文件` to `导入兑换内容`.
- Add an import mode segmented control or radio group with `输入文本` and `上传文件`.
- Default mode is `输入文本`.
- Category and template selectors remain required and shared by both modes.
- Text mode shows a textarea for multiline input.
- File mode shows the existing `el-upload` control.
- The generate button validates the active mode before sending the request.

Submitted form data:

- Always send `category_id` and `template_id`.
- In text mode send `text`.
- In file mode send `file`.

## Result Behavior

The current import result area remains unchanged. It still shows total valid lines, inserted rows, invalid row count, generated CDKs, and invalid row messages.

After a successful import:

- Refresh the redeem item list.
- Clear the active input source.
- Keep the selected category and template so the user can continue importing with the same setup.

## Testing

Backend tests will cover the behavior because the generation and validation rules live there:

- Text import with two non-empty lines creates two redeem items and two CDKs using the selected template.
- Text import ignores blank lines.
- Text import rejects empty or whitespace-only text with `请输入文本内容`.
- File import still works through the existing entry point.

Frontend verification will use `npm run build` and a browser check against `http://localhost:3000` after rebuilding `frontend/dist`.

## Scope

This design does not change template rendering, CDK generation, category permissions, team visibility, import history schema, or the manual edit dialog for existing redeem items.
