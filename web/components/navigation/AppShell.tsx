"use client";

import { AppShell as MantineAppShell, Burger, Group, Button, Text, Avatar, Menu, Stack } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import Link from "next/link";
import { useAuth } from "@/components/auth/AuthProvider";

export default function AppShell({ children }: { children: React.ReactNode }) {
  const [opened, { toggle }] = useDisclosure();
  const { user, logout } = useAuth();

  return (
    <MantineAppShell
      header={{ height: 60 }}
      navbar={{ width: 300, breakpoint: "sm", collapsed: { mobile: !opened } }}
      padding="md"
    >
      <MantineAppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Group>
            <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
            <Text component={Link} href="/" fw={700} size="lg" data-testid="app-logo">Call Booking</Text>
          </Group>
          {user && (
            <Menu>
              <Menu.Target>
                <Group gap="xs" style={{ cursor: "pointer" }} data-testid="user-menu">
                  <Avatar size="sm" color="blue">{user.name.charAt(0).toUpperCase()}</Avatar>
                  <Text size="sm" visibleFrom="sm" data-testid="user-name">{user.name}</Text>
                </Group>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item onClick={logout} color="red" data-testid="logout-button">Выйти</Menu.Item>
              </Menu.Dropdown>
            </Menu>
          )}
        </Group>
      </MantineAppShell.Header>

      <MantineAppShell.Navbar p="md">
        <Stack gap="xs">
          <Button component={Link} href="/" variant="subtle" justify="start" data-testid="nav-users">Каталог пользователей</Button>
          <Button component={Link} href="/my/schedule" variant="subtle" justify="start" data-testid="nav-schedule">Моё расписание</Button>
          <Button component={Link} href="/my/groups" variant="subtle" justify="start" data-testid="nav-groups">Мои группы</Button>
          <Button component={Link} href="/my/bookings" variant="subtle" justify="start" data-testid="nav-bookings">Мои бронирования</Button>
        </Stack>
      </MantineAppShell.Navbar>

      <MantineAppShell.Main>{children}</MantineAppShell.Main>
    </MantineAppShell>
  );
}
