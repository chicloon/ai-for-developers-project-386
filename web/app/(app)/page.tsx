"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Paper, Title, Stack, Card, Group, Text, Button, Loader, Center } from "@mantine/core";
import { User, getUsers } from "@/lib/api";

export default function UsersPage() {
  const router = useRouter();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadUsers();
  }, []);

  const loadUsers = async () => {
    try {
      setLoading(true);
      const data = await getUsers();
      setUsers(data.users);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <Center h="50vh" data-testid="users-loading">
        <Loader />
      </Center>
    );
  }

  return (
    <Stack gap="md" data-testid="users-page">
      <Title order={2} data-testid="users-title">Каталог пользователей</Title>
      {users.length === 0 ? (
        <Text c="dimmed" data-testid="users-empty">Пока нет доступных пользователей</Text>
      ) : (
        users.map((user) => (
          <Card key={user.id} withBorder data-testid={`user-card-${user.id}`}>
            <Group justify="space-between">
              <div>
                <Text fw={500} data-testid={`user-name-${user.id}`}>{user.name}</Text>
                <Text size="sm" c="dimmed" data-testid={`user-email-${user.id}`}>{user.email}</Text>
              </div>
              <Button 
                onClick={() => router.push(`/users/${user.id}`)} 
                variant="light"
                data-testid={`user-book-button-${user.id}`}
              >
                Записаться
              </Button>
            </Group>
          </Card>
        ))
      )}
    </Stack>
  );
}
