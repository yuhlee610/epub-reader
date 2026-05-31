export function Notification({message, error}) {
  if (error) {
    return (
      <div className="error-message" role="alert">
        {error}
      </div>
    );
  }

  if (!message) {
    return null;
  }

  return (
    <div className="status-message" role="status">
      {message}
    </div>
  );
}
