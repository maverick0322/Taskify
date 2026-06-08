import { type FormEvent, useState } from "react";

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
      const message =
        error instanceof Error
          ? error.message
          : "No pudimos completar la autenticacion.";
      setErrorMessage(message);
    } finally {
      setIsLoading(false);
    }
  };

  const toggleMode = () => {
    setIsLogin((currentValue) => !currentValue);
    setErrorMessage("");
  };

  return (
    <main className="flex min-h-screen flex-col items-center justify-center bg-slate-50 px-4 py-10">
      <Card className="w-full max-w-md border-slate-200 shadow-xl">
        <CardHeader className="space-y-2 text-center">
          <CardTitle className="text-2xl font-semibold tracking-normal text-slate-950">
            {title}
          </CardTitle>
          <CardDescription className="text-sm text-slate-500">
            {description}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-5" onSubmit={handleSubmit}>
            {!isLogin ? (
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="firstName">Nombre</Label>
                  <Input
                    autoComplete="given-name"
                    disabled={isLoading}
                    id="firstName"
                    onChange={(event) => setFirstName(event.target.value)}
                    required={!isLogin}
                    value={firstName}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="lastName">Apellido</Label>
                  <Input
                    autoComplete="family-name"
                    disabled={isLoading}
                    id="lastName"
                    onChange={(event) => setLastName(event.target.value)}
                    required={!isLogin}
                    value={lastName}
                  />
                </div>
              </div>
            ) : null}

            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                autoComplete="email"
                disabled={isLoading}
                id="email"
                onChange={(event) => setEmail(event.target.value)}
                placeholder="tu@email.com"
                required
                type="email"
                value={email}
              />
            </div>

            {!isLogin ? (
              <div className="space-y-2">
                <Label htmlFor="birthDate">Fecha de nacimiento</Label>
                <Input
                  disabled={isLoading}
                  id="birthDate"
                  onChange={(event) => setBirthDate(event.target.value)}
                  required={!isLogin}
                  type="date"
                  value={birthDate}
                />
              </div>
            ) : null}

            <div className="space-y-2">
              <Label htmlFor="password">Contrasena</Label>
              <Input
                autoComplete={isLogin ? "current-password" : "new-password"}
                disabled={isLoading}
                id="password"
                onChange={(event) => setPassword(event.target.value)}
                required
                type="password"
                value={password}
              />
            </div>

            {errorMessage ? (
              <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm font-medium text-red-700">
                {errorMessage}
              </p>
            ) : null}

            <Button className="w-full" disabled={isLoading} type="submit">
              {isLoading ? loadingLabel : submitLabel}
            </Button>
          </form>

          <Button
            className="mt-4 w-full text-slate-600"
            disabled={isLoading}
            onClick={toggleMode}
            type="button"
            variant="ghost"
          >
            {isLogin ? "No tienes cuenta? Registrate" : "Ya tienes cuenta? Inicia sesion"}
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
