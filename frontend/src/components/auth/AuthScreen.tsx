import { type FormEvent, useState } from "react";
import {
  AlertCircle,
  CalendarDays,
  Eye,
  EyeOff,
  Loader2,
  Lock,
  Mail,
  User,
} from "lucide-react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { getFriendlyErrorMessage } from "@/services/api";
import { login, register } from "@/services/authService";
import { useAuthStore } from "@/store/useAuthStore";

export function AuthScreen() {
  const setAuthenticatedSession = useAuthStore((state) => state.login);
  const [isLogin, setIsLogin] = useState(true);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [birthDate, setBirthDate] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  const title = isLogin ? "Iniciar sesion" : "Crear cuenta";
  const description = isLogin
    ? "Accede a tu tablero de trabajo."
    : "Crea tu usuario para empezar a organizar tareas.";
  const submitLabel = isLogin ? "Iniciar sesion" : "Crear cuenta";
  const loadingLabel = isLogin ? "Ingresando..." : "Creando cuenta...";

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setErrorMessage("");
    setIsLoading(true);

    try {
      if (!isLogin) {
        await register({
          email,
          password,
          firstName,
          lastName,
          birthDate,
        });
      }

      const tokenPair = await login({ email, password });
      localStorage.setItem("refreshToken", tokenPair.refreshToken);
      setAuthenticatedSession(tokenPair.accessToken);
    } catch (error) {
      setErrorMessage(
        getFriendlyErrorMessage(
          error,
          "No pudimos completar la autenticacion.",
        ),
      );
    } finally {
      setIsLoading(false);
    }
  };

  const toggleMode = () => {
    setIsLogin((currentValue) => !currentValue);
    setErrorMessage("");
  };

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 py-10">
      <div
        className="pointer-events-none fixed inset-0 -z-10 opacity-40"
        style={{
          backgroundImage:
            "radial-gradient(circle at 20% 30%, oklch(0.75 0.12 250 / 0.25) 0%, transparent 50%), radial-gradient(circle at 80% 70%, oklch(0.65 0.15 220 / 0.2) 0%, transparent 50%)",
        }}
      />

      <Card className="w-full max-w-md border border-border/60 p-0 shadow-2xl ring-0">
        <CardHeader className="space-y-3 p-6 pb-6 text-center">
          <div className="mx-auto mb-1 flex size-12 items-center justify-center rounded-xl bg-primary shadow-md">
            <Lock className="size-5 text-primary-foreground" />
          </div>
          <div>
            <CardTitle className="text-2xl font-semibold tracking-tight text-foreground">
              {title}
            </CardTitle>
            <CardDescription className="mt-1 text-sm text-muted-foreground">
              {description}
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="px-6 pb-6">
          <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
            <div
              className={cn(
                "grid overflow-hidden transition-all duration-300 ease-in-out",
                !isLogin
                  ? "grid-rows-[1fr] opacity-100"
                  : "grid-rows-[0fr] opacity-0",
              )}
            >
              <div className="overflow-hidden">
                <div className="flex flex-col gap-4 pb-1 pt-0.5">
                  <div className="grid gap-3 sm:grid-cols-2">
                    <div className="flex flex-col gap-1.5">
                      <Label htmlFor="firstName" className="text-sm font-medium">
                        Nombre
                      </Label>
                      <div className="relative">
                        <User className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                        <Input
                          autoComplete="given-name"
                          className="pl-9"
                          disabled={isLoading}
                          id="firstName"
                          onChange={(event) => setFirstName(event.target.value)}
                          placeholder="Juan"
                          required={!isLogin}
                          value={firstName}
                        />
                      </div>
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <Label htmlFor="lastName" className="text-sm font-medium">
                        Apellido
                      </Label>
                      <div className="relative">
                        <User className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                        <Input
                          autoComplete="family-name"
                          className="pl-9"
                          disabled={isLoading}
                          id="lastName"
                          onChange={(event) => setLastName(event.target.value)}
                          placeholder="Garcia"
                          required={!isLogin}
                          value={lastName}
                        />
                      </div>
                    </div>
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="birthDate" className="text-sm font-medium">
                      Fecha de nacimiento
                    </Label>
                    <div className="relative">
                      <CalendarDays className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        className="pl-9"
                        disabled={isLoading}
                        id="birthDate"
                        onChange={(event) => setBirthDate(event.target.value)}
                        required={!isLogin}
                        type="date"
                        value={birthDate}
                      />
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="email" className="text-sm font-medium">
                Correo electronico
              </Label>
              <div className="relative">
                <Mail className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  autoComplete="email"
                  className="pl-9"
                  disabled={isLoading}
                  id="email"
                  onChange={(event) => setEmail(event.target.value)}
                  placeholder="tu@email.com"
                  required
                  type="email"
                  value={email}
                />
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="password" className="text-sm font-medium">
                Contrasena
              </Label>
              <div className="relative">
                <Lock className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  autoComplete={isLogin ? "current-password" : "new-password"}
                  className="pl-9 pr-10"
                  disabled={isLoading}
                  id="password"
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="********"
                  required
                  type={showPassword ? "text" : "password"}
                  value={password}
                />
                <button
                  type="button"
                  aria-label={
                    showPassword ? "Ocultar contrasena" : "Mostrar contrasena"
                  }
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                  onClick={() => setShowPassword((currentValue) => !currentValue)}
                  tabIndex={-1}
                >
                  {showPassword ? (
                    <EyeOff className="size-4" />
                  ) : (
                    <Eye className="size-4" />
                  )}
                </button>
              </div>
            </div>

            {errorMessage ? (
              <Alert variant="destructive" className="py-2.5">
                <AlertCircle className="size-4" />
                <AlertDescription className="text-sm">
                  {errorMessage}
                </AlertDescription>
              </Alert>
            ) : null}

            <Button
              className="mt-1 w-full gap-2 font-semibold shadow-sm"
              disabled={isLoading}
              size="lg"
              type="submit"
            >
              {isLoading ? <Loader2 className="size-4 animate-spin" /> : null}
              {isLoading ? loadingLabel : submitLabel}
            </Button>
          </form>

          <div className="relative my-5">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-border" />
            </div>
            <div className="relative flex justify-center text-xs">
              <span className="bg-card px-3 text-muted-foreground">o</span>
            </div>
          </div>

          <Button
            className="w-full font-medium text-muted-foreground hover:text-foreground"
            disabled={isLoading}
            onClick={toggleMode}
            type="button"
            variant="outline"
          >
            {isLogin
              ? "No tienes cuenta? Registrate"
              : "Ya tienes cuenta? Inicia sesion"}
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
