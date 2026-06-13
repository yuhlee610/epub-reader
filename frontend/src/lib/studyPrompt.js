const DEFAULT_STUDY_INSTRUCTIONS = [
  'Translate naturally into Vietnamese with the book context in mind.',
  'List important vocabulary from the selected text with short meanings.',
  'Explain useful grammar patterns briefly.',
  'Add any nuance needed to understand the passage.',
].join('\n');

const MAX_SELECTED_TEXT_LENGTH = 6000;

export function buildBookAwareStudyPrompt({book, chapter, text}) {
  const selectedText = cleanPromptText(text);
  const bookTitle = cleanPromptText(book?.title) || 'the current book';
  const author = cleanPromptText(book?.author);
  const chapterTitle = cleanPromptText(chapter?.title);
  const activePrompt = cleanPromptText(book?.prompt?.customPrompt) || DEFAULT_STUDY_INSTRUCTIONS;
  const preparedText = selectedText.length > MAX_SELECTED_TEXT_LENGTH
    ? `${selectedText.slice(0, MAX_SELECTED_TEXT_LENGTH).trim()}\n\n[Selected text truncated for prompt length.]`
    : selectedText;

  return [
    'You are helping me study and translate a passage from a book.',
    '',
    'Book context:',
    `- Title: ${bookTitle}`,
    `- Author: ${author || 'Unknown or not available'}`,
    `- Chapter: ${chapterTitle || 'Current chapter'}`,
    '',
    'Active per-book instructions:',
    activePrompt,
    '',
    'Selected passage:',
    preparedText || '[No selected passage was provided.]',
    '',
    'Response requirements:',
    '- Provide a natural translation that respects the book context.',
    '- Explain important vocabulary from the selected passage.',
    '- Explain useful grammar patterns from the selected passage.',
    '- Keep the explanation concise and useful for language learning.',
  ].join('\n');
}

function cleanPromptText(value) {
  return String(value || '').replace(/\s+/g, ' ').trim();
}
