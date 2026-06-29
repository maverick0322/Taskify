import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useDesktopUpdater } from "@/hooks/useDesktopUpdater";

function formatPublishedDate(date?: string) {
  if (!date) {
    return null;
  }

  const parsedDate = new Date(date);
  if (Number.isNaN(parsedDate.getTime())) {
    return null;
  }

  return parsedDate.toLocaleDateString("es-MX", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}

export function UpdateDialog() {
  const {
    availableUpdate,
    contentLength,
    dismissUpdate,
    installUpdate,
    isDialogOpen,
    progressPercent,
    stage,
  } = useDesktopUpdater();

  const publishedDate = formatPublishedDate(availableUpdate?.date);
  const isBusy = stage === "checking" || stage === "downloading" || stage === "installing";

  return (
    <Dialog
      open={isDialogOpen}
      onOpenChange={(open) => {
        if (!open) {
          dismissUpdate();
        }
      }}
    >
      <DialogContent className="sm:max-w-md" showCloseButton={!isBusy}>
        <DialogHeader>
          <DialogTitle>Nueva versión disponible</DialogTitle>
          <DialogDescription>
            Hay una actualización de Taskify lista para descargar.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-lg border border-border/70 bg-muted/40 p-4">
            <div className="flex items-center justify-between gap-3 text-sm">
              <span className="text-muted-foreground">Versión actual</span>
              <span className="font-medium text-foreground">
                {availableUpdate?.currentVersion ?? "-"}
              </span>
            </div>
            <div className="mt-2 flex items-center justify-between gap-3 text-sm">
              <span className="text-muted-foreground">Nueva versión</span>
              <span className="font-medium text-foreground">
                {availableUpdate?.version ?? "-"}
              </span>
            </div>
            {publishedDate ? (
              <div className="mt-2 flex items-center justify-between gap-3 text-sm">
                <span className="text-muted-foreground">Publicada</span>
                <span className="font-medium text-foreground">{publishedDate}</span>
              </div>
            ) : null}
          </div>

          {availableUpdate?.body ? (
            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">Notas</p>
              <p className="whitespace-pre-line text-sm text-muted-foreground">
                {availableUpdate.body}
              </p>
            </div>
          ) : null}

          {stage === "downloading" || stage === "installing" ? (
            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">
                {stage === "downloading"
                  ? "Descargando actualización..."
                  : "Instalando actualización..."}
              </p>
              <div className="h-2 overflow-hidden rounded-full bg-muted">
                <div
                  className="h-full rounded-full bg-primary transition-all"
                  style={{
                    width:
                      progressPercent === null
                        ? "100%"
                        : `${progressPercent}%`,
                  }}
                />
              </div>
              <p className="text-xs text-muted-foreground">
                {stage === "downloading" && progressPercent !== null
                  ? `${progressPercent}% descargado`
                  : contentLength
                    ? "Procesando paquete de actualización..."
                    : "Preparando la actualización..."}
              </p>
            </div>
          ) : null}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={dismissUpdate} disabled={isBusy}>
            Más tarde
          </Button>
          <Button onClick={() => void installUpdate()} disabled={isBusy}>
            {stage === "downloading" || stage === "installing"
              ? "Actualizando..."
              : "Actualizar ahora"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
