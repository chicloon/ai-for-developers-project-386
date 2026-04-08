"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Paper, Title, TextInput, PasswordInput, Button, Stack, Text, Anchor } from "@mantine/core";
import { login } from "@/lib/api";
import { useAuth } from "@/components/auth/AuthProvider";

export default function LoginPage() {
  const router = useRouter();
  const { login: authLogin } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const response = await login({ email, password });
      authLogin(response.token, response.user);
      router.push("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка входа");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Paper p="xl" maw={400} mx="auto" mt={100} withBorder data-testid="login-page">
      <Title order={2} mb="lg" ta="center" data-testid="login-title">Вход</Title>
      <form onSubmit={handleSubmit} data-testid="login-form">
        <Stack>
          <TextInput 
            label="Email" 
            type="email" 
            value={email} 
            onChange={(e) => setEmail(e.target.value)} 
            required 
            data-testid="login-email-input"
          />
          <PasswordInput 
            label="Пароль" 
            value={password} 
            onChange={(e) => setPassword(e.target.value)} 
            required 
            data-testid="login-password-input"
          />
          {error && <Text c="red" size="sm" data-testid="login-error">{error}</Text>}
          <Button type="submit" loading={loading} fullWidth data-testid="login-submit-button">
            Войти
          </Button>
          <Text ta="center" size="sm">
            Нет аккаунта? <Anchor href="/register" data-testid="login-register-link">Зарегистрироваться</Anchor>
          </Text>
        </Stack>
      </form>
    </Paper>
  );
}
