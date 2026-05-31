// formatImportedAt renders backend timestamps with the user's locale settings.
export function formatImportedAt(value) {
  if (!value) {
    return 'Unknown import date';
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}

// progressLabel formats the currently available persisted reading progress.
export function progressLabel(book) {
  if (!book?.progress?.chapterIndex) {
    return '0%';
  }

  return `${Math.max(0, book.progress.chapterIndex)}%`;
}

// coverTitle keeps generated placeholder covers readable for books without art.
export function coverTitle(title) {
  const words = (title || 'Untitled Book').split(/\s+/).filter(Boolean);
  return words.slice(0, 4).join(' ');
}

// formatImportError turns backend import failures into UI copy a reader can act on.
export function formatImportError(err) {
  const message = err?.message ?? String(err);
  const lowerMessage = message.toLowerCase();

  if (lowerMessage.includes('invalid epub file')) {
    return [
      'That file does not look like a valid EPUB.',
      'Choose a .epub file and try again.',
    ].join(' ');
  }
  if (lowerMessage.includes('copy epub file')) {
    return 'The book could not be copied into the local library. Check file access and try again.';
  }
  if (lowerMessage.includes('open epub picker')) {
    return 'The file picker could not be opened. Try importing again.';
  }

  return message;
}

// fileNameLabel shows a stable filename without exposing the full local path.
export function fileNameLabel(book) {
  return book?.originalFileName || 'Managed EPUB';
}
