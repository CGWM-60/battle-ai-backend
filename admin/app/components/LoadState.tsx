export function LoadingState() {
  return <div className="panel muted-panel">Chargement des donnees admin...</div>;
}

export function ErrorState({ message }: { message: string }) {
  return <div className="alert error">Impossible de charger les donnees: {message}</div>;
}
