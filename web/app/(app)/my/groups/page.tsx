"use client";

import { useEffect, useState } from "react";
import {
  Paper,
  Title,
  Stack,
  Accordion,
  Button,
  Loader,
  Center,
  Text,
  Modal,
  TextInput,
  Group,
  ActionIcon,
  Badge,
  Select,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconTrash, IconUserPlus, IconEdit } from "@tabler/icons-react";
import {
  VisibilityGroup,
  GroupMember,
  CreateGroupRequest,
  getMyGroups,
  createGroup,
  updateGroup,
  deleteGroup,
  getGroupMembers,
  addGroupMember,
  removeGroupMember,
} from "@/lib/api";
import { useAuth } from "@/components/auth/AuthProvider";

const VISIBILITY_LEVELS = [
  { value: "family", label: "Семья" },
  { value: "work", label: "Работа" },
  { value: "friends", label: "Друзья" },
  { value: "public", label: "Публичная" },
];

export default function MyGroupsPage() {
  const { user: currentUser } = useAuth();
  const [groups, setGroups] = useState<VisibilityGroup[]>([]);
  const [groupMembers, setGroupMembers] = useState<Record<string, GroupMember[]>>({});
  const [loading, setLoading] = useState(true);
  const [createOpened, { open: openCreate, close: closeCreate }] = useDisclosure(false);
  const [editOpened, { open: openEdit, close: closeEdit }] = useDisclosure(false);
  const [addMemberOpened, { open: openAddMember, close: closeAddMember }] = useDisclosure(false);
  const [submitting, setSubmitting] = useState(false);
  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null);

  // Form state
  const [groupName, setGroupName] = useState("");
  const [visibilityLevel, setVisibilityLevel] = useState<string | null>("public");
  const [memberEmail, setMemberEmail] = useState("");

  useEffect(() => {
    loadGroups();
  }, []);

  const loadGroups = async () => {
    try {
      setLoading(true);
      const data = await getMyGroups();
      setGroups(data.groups);

      // Load members for each group
      const membersMap: Record<string, GroupMember[]> = {};
      for (const group of data.groups) {
        try {
          const membersData = await getGroupMembers(group.id);
          membersMap[group.id] = membersData.members;
        } catch (e) {
          console.error(e);
        }
      }
      setGroupMembers(membersMap);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setGroupName("");
    setVisibilityLevel("public");
    setSelectedGroupId(null);
  };

  const handleOpenCreate = () => {
    resetForm();
    openCreate();
  };

  const handleOpenEdit = (group: VisibilityGroup) => {
    setSelectedGroupId(group.id);
    setGroupName(group.name);
    setVisibilityLevel(group.visibilityLevel);
    openEdit();
  };

  const handleCreateGroup = async () => {
    if (!groupName.trim() || !visibilityLevel) return;
    try {
      setSubmitting(true);
      const data: CreateGroupRequest = {
        name: groupName.trim(),
        visibilityLevel: visibilityLevel as "family" | "work" | "friends" | "public",
      };
      await createGroup(data);
      closeCreate();
      resetForm();
      await loadGroups();
    } catch (e) {
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  const handleUpdateGroup = async () => {
    if (!selectedGroupId || !groupName.trim() || !visibilityLevel) return;
    try {
      setSubmitting(true);
      const data: CreateGroupRequest = {
        name: groupName.trim(),
        visibilityLevel: visibilityLevel as "family" | "work" | "friends" | "public",
      };
      await updateGroup(selectedGroupId, data);
      closeEdit();
      resetForm();
      await loadGroups();
    } catch (e) {
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteGroup = async (id: string) => {
    try {
      await deleteGroup(id);
      await loadGroups();
    } catch (e) {
      console.error(e);
    }
  };

  const handleAddMember = async () => {
    if (!selectedGroupId || !memberEmail.trim()) return;
    try {
      setSubmitting(true);
      await addGroupMember(selectedGroupId, { email: memberEmail.trim() });
      closeAddMember();
      setMemberEmail("");
      setSelectedGroupId(null);
      await loadGroups();
    } catch (e) {
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  const handleRemoveMember = async (groupId: string, memberId: string) => {
    try {
      await removeGroupMember(groupId, memberId);
      await loadGroups();
    } catch (e) {
      console.error(e);
    }
  };

  const openAddMemberModal = (groupId: string) => {
    setSelectedGroupId(groupId);
    setMemberEmail("");
    openAddMember();
  };

  if (loading) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Title order={2}>Мои группы</Title>
        <Button onClick={handleOpenCreate}>Создать группу</Button>
      </Group>

      {groups.length === 0 ? (
        <Text c="dimmed">У вас пока нет групп</Text>
      ) : (
        <Accordion>
          {groups.map((group) => {
            const members = groupMembers[group.id] || [];
            const isOwner = group.ownerId === currentUser?.id;
            const visibilityLabel = VISIBILITY_LEVELS.find(
              (v) => v.value === group.visibilityLevel
            )?.label;

            return (
              <Accordion.Item key={group.id} value={group.id}>
                <Accordion.Control>
                  <Group justify="space-between">
                    <Group gap="xs">
                      <Text fw={500}>{group.name}</Text>
                      <Badge color="blue" variant="light">
                        {visibilityLabel}
                      </Badge>
                    </Group>
                    {isOwner && <Badge color="blue">Владелец</Badge>}
                  </Group>
                </Accordion.Control>
                <Accordion.Panel>
                  <Stack gap="xs">
                    {members.length === 0 ? (
                      <Text c="dimmed" size="sm">
                        В группе пока нет участников
                      </Text>
                    ) : (
                      members.map((member) => (
                        <Paper key={member.id} withBorder p="xs">
                          <Group justify="space-between">
                            <div>
                              <Text size="sm" fw={500}>
                                {member.member.name}
                              </Text>
                              <Text size="xs" c="dimmed">
                                {member.member.email}
                              </Text>
                            </div>
                            {isOwner && member.member.id !== currentUser?.id && (
                              <ActionIcon
                                color="red"
                                size="sm"
                                onClick={() =>
                                  handleRemoveMember(group.id, member.id)
                                }
                              >
                                <IconTrash size={14} />
                              </ActionIcon>
                            )}
                          </Group>
                        </Paper>
                      ))
                    )}

                    {isOwner && (
                      <Group justify="space-between" mt="xs">
                        <Button
                          size="xs"
                          variant="light"
                          leftSection={<IconUserPlus size={14} />}
                          onClick={() => openAddMemberModal(group.id)}
                        >
                          Добавить участника
                        </Button>
                        <Group gap="xs">
                          <ActionIcon
                            color="blue"
                            onClick={() => handleOpenEdit(group)}
                          >
                            <IconEdit size={16} />
                          </ActionIcon>
                          <ActionIcon
                            color="red"
                            onClick={() => handleDeleteGroup(group.id)}
                          >
                            <IconTrash size={16} />
                          </ActionIcon>
                        </Group>
                      </Group>
                    )}
                  </Stack>
                </Accordion.Panel>
              </Accordion.Item>
            );
          })}
        </Accordion>
      )}

      {/* Create Group Modal */}
      <Modal opened={createOpened} onClose={closeCreate} title="Создать группу">
        <Stack gap="md">
          <TextInput
            label="Название группы"
            placeholder="Введите название"
            value={groupName}
            onChange={(e) => setGroupName(e.target.value)}
          />
          <Select
            label="Уровень видимости"
            value={visibilityLevel}
            onChange={setVisibilityLevel}
            data={VISIBILITY_LEVELS}
          />
          <Group justify="flex-end">
            <Button variant="default" onClick={closeCreate}>
              Отмена
            </Button>
            <Button
              onClick={handleCreateGroup}
              loading={submitting}
              disabled={!groupName.trim() || !visibilityLevel}
            >
              Создать
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Edit Group Modal */}
      <Modal opened={editOpened} onClose={closeEdit} title="Редактировать группу">
        <Stack gap="md">
          <TextInput
            label="Название группы"
            placeholder="Введите название"
            value={groupName}
            onChange={(e) => setGroupName(e.target.value)}
          />
          <Select
            label="Уровень видимости"
            value={visibilityLevel}
            onChange={setVisibilityLevel}
            data={VISIBILITY_LEVELS}
          />
          <Group justify="flex-end">
            <Button variant="default" onClick={closeEdit}>
              Отмена
            </Button>
            <Button
              onClick={handleUpdateGroup}
              loading={submitting}
              disabled={!groupName.trim() || !visibilityLevel}
            >
              Сохранить
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Add Member Modal */}
      <Modal
        opened={addMemberOpened}
        onClose={closeAddMember}
        title="Добавить участника"
      >
        <Stack gap="md">
          <TextInput
            label="Email пользователя"
            placeholder="user@example.com"
            value={memberEmail}
            onChange={(e) => setMemberEmail(e.target.value)}
          />
          <Group justify="flex-end">
            <Button variant="default" onClick={closeAddMember}>
              Отмена
            </Button>
            <Button
              onClick={handleAddMember}
              loading={submitting}
              disabled={!memberEmail.trim()}
            >
              Добавить
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
