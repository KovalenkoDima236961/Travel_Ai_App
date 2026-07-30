# Timeline

The trip detail page now wraps itinerary rendering in `SchedulePlanningWorkspace`.

The workspace provides synchronized Agenda, Timeline, and Calendar views over the same `Itinerary` object. In edit mode, visual schedule edits mutate the existing draft itinerary and are not saved until the user clicks the existing Save button.

Timeline behavior:

- Native drag/drop moves scheduled activity blocks within a day or between days when `timeline_drag_drop_enabled` is true.
- Keyboard-accessible controls move items earlier/later, change duration, move yesterday/tomorrow, or mark an item unscheduled.
- Travel and transfer-like items render as travel blocks and are treated as non-editable travel context.
- The unscheduled panel lists untimed items and lets editors assign them back into the schedule.
- Undo is local to the current edit session and applies before saving.

The read-only Agenda tab keeps the existing trip-detail timeline renderer so regeneration, comments, reactions, provider availability, and cost split controls remain intact.
