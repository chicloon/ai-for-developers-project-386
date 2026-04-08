"use client";

export const dynamic = 'force-dynamic';

import { useEffect, useState } from "react";
import {
  Paper,
  Title,
  Stack,
  Table,
  Button,
  Loader,
  Center,
  Text,
  Modal,
  TextInput,
  Select,
  Group,
  ActionIcon,
  Checkbox,
  Accordion,
  Badge,
  Switch,
  Alert,
  Tabs,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { TimeInput } from "@mantine/dates";
import { IconTrash, IconEdit, IconUserPlus, IconInfoCircle } from "@tabler/icons-react";
import {
  Schedule,
  VisibilityGroup,
  GroupMember,
  User,
  getMySchedules,
  createSchedule,
  updateSchedule,
  deleteSchedule,
  getMyGroups,
  getGroupMembers,
  addGroupMember,
  removeGroupMember,
  getMe,
  updateMe,
  CreateScheduleRequest,
} from "@/lib/api";
import { useAuth } from "@/components/auth/AuthProvider";

const DAYS_OF_WEEK = [
  { value: "1", label: "Понедельник" },
  { value: "2", label: "Вторник" },
  { value: "3", label: "Среда" },
  { value: "4", label: "Четверг" },
  { value: "5", label: "Пятница" },
  { value: "6", label: "Суббота" },
  { value: "0", label: "Воскресенье" },
];

// Fixed group order and display info
const GROUP_INFO: Record<string, { label: string; color: string; description: string }> = {
  family: { label: "Семья", color: "green", description: "Члены вашей семьи" },
  friends: { label: "Друзья", color: "blue", description: "Ваши друзья" },
  work: { label: "Работа", color: "orange", description: "Коллеги по работе" },
};

export default function MySchedulePage() {
  const { user: currentUser, setUser } = useAuth();

  // Loading states
  const [loading, setLoading] = useState(true);

  // Visibility/Groups state
  const [groups, setGroups] = useState<VisibilityGroup[]>([]);
  const [groupMembers, setGroupMembers] = useState<Record<string, GroupMember[]>>({});
  const [isPublic, setIsPublic] = useState(false);
  const [updatingPublic, setUpdatingPublic] = useState(false);

  // Schedule state
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [scheduleModalOpened, { open: openScheduleModal, close: closeScheduleModal }] = useDisclosure(false);
  const [editingScheduleId, setEditingScheduleId] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Member modal state
  const [addMemberOpened, { open: openAddMember, close: closeAddMember }] = useDisclosure(false);
  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null);
  const [memberEmail, setMemberEmail] = useState("");
  const [addingMember, setAddingMember] = useState(false);

  // Schedule form state
  const [type, setType] = useState<"recurring" | "one-time">("recurring");
  const [dayOfWeek, setDayOfWeek] = useState<string | null>("1");
  const [date, setDate] = useState<string>("");
  const [startTime, setStartTime] = useState<string>("09:00");
  const [endTime, setEndTime] = useState<string>("17:00");
  const [isBlocked, setIsBlocked] = useState<boolean>(false);
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>([]);

  // Tab state
  const [activeTab, setActiveTab] = useState<string | null>("schedule");

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);

      // Load current user for isPublic status
      const me = await getMe();
      setIsPublic(me.isPublic);
      if (setUser) {
        setUser(me);
      }

      // Load groups
      const groupsData = await getMyGroups();
      setGroups(groupsData.groups);

      // Load members for each group
      const membersMap: Record<string, GroupMember[]> = {};
      for (const group of groupsData.groups) {
        try {
          const membersData = await getGroupMembers(group.id);
          membersMap[group.id] = membersData.members;
        } catch (e) {
          console.error(e);
        }
      }
      setGroupMembers(membersMap);

      // Load schedules
      const schedulesData = await getMySchedules();
      setSchedules(schedulesData.schedules);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  // Visibility handlers
  const handleTogglePublic = async (checked: boolean) => {
    try {
      setUpdatingPublic(true);
      const updated = await updateMe({ isPublic: checked });
      setIsPublic(updated.isPublic);
      if (setUser) {
        setUser(updated);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setUpdatingPublic(false);
    }
  };

  // Group member handlers
  const handleAddMember = async () => {
    if (!selectedGroupId || !memberEmail.trim()) return;
    try {
      setAddingMember(true);
      await addGroupMember(selectedGroupId, { email: memberEmail.trim() });
      closeAddMember();
      setMemberEmail("");
      setSelectedGroupId(null);
      await loadData();
    } catch (e) {
      console.error(e);
    } finally {
      setAddingMember(false);
    }
  };

  const handleRemoveMember = async (groupId: string, memberId: string) => {
    try {
      await removeGroupMember(groupId, memberId);
      await loadData();
    } catch (e) {
      console.error(e);
    }
  };

  const openAddMemberModal = (groupId: string) => {
    setSelectedGroupId(groupId);
    setMemberEmail("");
    openAddMember();
  };

  // Schedule handlers
  const handleOpenCreateSchedule = () => {
    setEditingScheduleId(null);
    resetScheduleForm();
    openScheduleModal();
  };

  const handleOpenEditSchedule = (schedule: Schedule) => {
    setEditingScheduleId(schedule.id);
    setType(schedule.type);
    setDayOfWeek(schedule.dayOfWeek?.toString() || "1");
    setDate(schedule.date || "");
    setStartTime(schedule.startTime);
    setEndTime(schedule.endTime);
    setIsBlocked(schedule.isBlocked);
    setSelectedGroupIds(schedule.groupIds || []);
    openScheduleModal();
  };

  const handleSubmitSchedule = async () => {
    try {
      setSubmitting(true);
      const data: CreateScheduleRequest = {
        type,
        dayOfWeek: type === "recurring" ? parseInt(dayOfWeek || "1") : undefined,
        date: type === "one-time" ? date : undefined,
        startTime,
        endTime,
        isBlocked,
        groupIds: selectedGroupIds.length > 0 ? selectedGroupIds : undefined,
      };

      if (editingScheduleId) {
        await updateSchedule(editingScheduleId, data);
      } else {
        await createSchedule(data);
      }
      closeScheduleModal();
      resetScheduleForm();
      await loadData();
    } catch (e) {
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteSchedule = async (id: string) => {
    try {
      await deleteSchedule(id);
      await loadData();
    } catch (e) {
      console.error(e);
    }
  };

  const resetScheduleForm = () => {
    setType("recurring");
    setDayOfWeek("1");
    setDate("");
    setStartTime("09:00");
    setEndTime("17:00");
    setIsBlocked(false);
    setSelectedGroupIds([]);
    setEditingScheduleId(null);
  };

  // Sort groups in fixed order
  const sortedGroups = [...groups].sort((a, b) => {
    const order = ["family", "friends", "work"];
    return order.indexOf(a.visibilityLevel) - order.indexOf(b.visibilityLevel);
  });

  if (loading) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }

  return (
    <Stack gap="md" data-testid="schedule-page">
      <Title order={2}>Моё расписание</Title>

      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.List>
          <Tabs.Tab value="schedule" data-testid="tab-schedule">Расписания</Tabs.Tab>
          <Tabs.Tab value="visibility" data-testid="tab-visibility">Настройки видимости</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="schedule" pt="md">
          <Stack gap="md">
            <Group justify="space-between">
              <Text size="sm" c="dimmed">
                Управляйте временем, когда вы доступны для записи
              </Text>
              <Button onClick={handleOpenCreateSchedule} data-testid="schedule-add-button">
                Добавить расписание
              </Button>
            </Group>

            {schedules.length === 0 ? (
              <Text c="dimmed" data-testid="schedule-empty">
                У вас пока нет настроенных расписаний
              </Text>
            ) : (
              <Paper withBorder data-testid="schedule-table">
                <Table>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Тип</Table.Th>
                      <Table.Th>День/дата</Table.Th>
                      <Table.Th>Время</Table.Th>
                      <Table.Th>Статус</Table.Th>
                      <Table.Th>Видимость</Table.Th>
                      <Table.Th></Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {schedules.map((schedule) => (
                      <Table.Tr key={schedule.id} data-testid={`schedule-row-${schedule.id}`}>
                        <Table.Td data-testid={`schedule-type-${schedule.id}`}>
                          {schedule.type === "recurring" ? "Повторяющееся" : "Разовое"}
                        </Table.Td>
                        <Table.Td data-testid={`schedule-day-${schedule.id}`}>
                          {schedule.type === "recurring"
                            ? DAYS_OF_WEEK.find((d) => d.value === String(schedule.dayOfWeek))?.label
                            : schedule.date}
                        </Table.Td>
                        <Table.Td data-testid={`schedule-time-${schedule.id}`}>
                          {schedule.startTime} - {schedule.endTime}
                        </Table.Td>
                        <Table.Td data-testid={`schedule-status-${schedule.id}`}>
                          {schedule.isBlocked ? (
                            <Text c="red" size="sm">Заблокировано</Text>
                          ) : (
                            <Text c="green" size="sm">Доступно</Text>
                          )}
                        </Table.Td>
                        <Table.Td>
                          {schedule.groupIds && schedule.groupIds.length > 0 ? (
                            <Group gap="xs">
                              {schedule.groupIds.map((gid) => {
                                const group = groups.find((g) => g.id === gid);
                                const info = group ? GROUP_INFO[group.visibilityLevel] : null;
                                return (
                                  <Badge key={gid} size="sm" color={info?.color || "blue"} variant="light">
                                    {info?.label || "Группа"}
                                  </Badge>
                                );
                              })}
                            </Group>
                          ) : (
                            <Text size="sm" c="dimmed">
                              {isPublic ? "Общее (всем)" : "Не видно"}
                            </Text>
                          )}
                        </Table.Td>
                        <Table.Td>
                          <Group gap="xs">
                            <ActionIcon
                              color="blue"
                              onClick={() => handleOpenEditSchedule(schedule)}
                              data-testid={`schedule-edit-${schedule.id}`}
                            >
                              <IconEdit size={16} />
                            </ActionIcon>
                            <ActionIcon
                              color="red"
                              onClick={() => handleDeleteSchedule(schedule.id)}
                              data-testid={`schedule-delete-${schedule.id}`}
                            >
                              <IconTrash size={16} />
                            </ActionIcon>
                          </Group>
                        </Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              </Paper>
            )}
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="visibility" pt="md">
          <Stack gap="md">
            {/* Public Profile Toggle */}
            <Paper withBorder p="md">
              <Stack gap="sm">
                <Group justify="space-between" align="flex-start">
                  <div>
                    <Text fw={500} size="lg">Публичный профиль</Text>
                    <Text size="sm" c="dimmed">
                      Когда включено, любой пользователь может видеть ваши общие расписания
                    </Text>
                  </div>
                  <Switch
                    checked={isPublic}
                    onChange={(e) => handleTogglePublic(e.currentTarget.checked)}
                    disabled={updatingPublic}
                    size="md"
                  />
                </Group>

                {isPublic && (
                  <Alert icon={<IconInfoCircle size={16} />} color="blue" variant="light">
                    <Text size="sm">
                      Ваши расписания без привязки к группам видны всем пользователям. Расписания с привязкой к группам видны только их участникам.
                    </Text>
                  </Alert>
                )}

                {!isPublic && (
                  <Alert icon={<IconInfoCircle size={16} />} color="gray" variant="light">
                    <Text size="sm">
                      Только участники ваших групп могут видеть ваши расписания. Общие расписания (без групп) не видны никому.
                    </Text>
                  </Alert>
                )}
              </Stack>
            </Paper>

            {/* Fixed Groups */}
            <Title order={3} mt="md">Мои группы</Title>

            {sortedGroups.length === 0 ? (
              <Alert color="yellow">
                <Text>Группы не найдены. Они должны быть созданы автоматически при регистрации.</Text>
              </Alert>
            ) : (
              <Accordion>
                {sortedGroups.map((group) => {
                  const members = groupMembers[group.id] || [];
                  const info = GROUP_INFO[group.visibilityLevel];

                  return (
                    <Accordion.Item key={group.id} value={group.id}>
                      <Accordion.Control>
                        <Group justify="space-between">
                          <Group gap="xs">
                            <Text fw={500}>{info?.label || group.name}</Text>
                            <Badge color={info?.color || "blue"} variant="light">
                              {members.length} участников
                            </Badge>
                          </Group>
                        </Group>
                      </Accordion.Control>
                      <Accordion.Panel>
                        <Stack gap="xs">
                          <Text size="sm" c="dimmed" mb="xs">
                            {info?.description}
                          </Text>

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
                                  {member.member.id !== currentUser?.id && (
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

                          <Button
                            size="xs"
                            variant="light"
                            leftSection={<IconUserPlus size={14} />}
                            onClick={() => openAddMemberModal(group.id)}
                            mt="xs"
                          >
                            Добавить участника
                          </Button>
                        </Stack>
                      </Accordion.Panel>
                    </Accordion.Item>
                  );
                })}
              </Accordion>
            )}
          </Stack>
        </Tabs.Panel>
      </Tabs>

      {/* Schedule Modal */}
      <Modal
        opened={scheduleModalOpened}
        onClose={closeScheduleModal}
        title={editingScheduleId ? "Редактировать расписание" : "Добавить расписание"}
        size="lg"
        data-testid="schedule-modal"
      >
        <Stack gap="md">
          <Select
            label="Тип расписания"
            value={type}
            onChange={(value) => setType(value as "recurring" | "one-time")}
            data={[
              { value: "recurring", label: "Повторяющееся" },
              { value: "one-time", label: "Разовое" },
            ]}
            data-testid="schedule-type-select"
          />

          {type === "recurring" ? (
            <Select
              label="День недели"
              value={dayOfWeek}
              onChange={setDayOfWeek}
              data={DAYS_OF_WEEK}
              data-testid="schedule-day-select"
            />
          ) : (
            <TextInput
              label="Дата"
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              data-testid="schedule-date-input"
            />
          )}

          <Group grow>
            <TimeInput
              label="Начало"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              data-testid="schedule-start-time"
            />
            <TimeInput
              label="Конец"
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
              data-testid="schedule-end-time"
            />
          </Group>

          <Checkbox
            label="Заблокировать (недоступно для бронирования)"
            checked={isBlocked}
            onChange={(e) => setIsBlocked(e.currentTarget.checked)}
            data-testid="schedule-blocked-checkbox"
          />

          <Checkbox.Group
            label="Видимость для групп"
            description="Если не выбрано ни одной группы, расписание будет общим (видно всем, если включен публичный профиль)"
            value={selectedGroupIds}
            onChange={setSelectedGroupIds}
          >
            <Stack gap="xs" mt="xs">
              {sortedGroups.map((group) => {
                const info = GROUP_INFO[group.visibilityLevel];
                return (
                  <Checkbox
                    key={group.id}
                    value={group.id}
                    label={
                      <Group gap="xs">
                        <Badge size="sm" color={info?.color || "blue"} variant="light">
                          {info?.label || group.name}
                        </Badge>
                        <Text size="sm">{info?.description}</Text>
                      </Group>
                    }
                  />
                );
              })}
            </Stack>
          </Checkbox.Group>

          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={closeScheduleModal} data-testid="schedule-cancel-button">
              Отмена
            </Button>
            <Button onClick={handleSubmitSchedule} loading={submitting} data-testid="schedule-submit-button">
              {editingScheduleId ? "Сохранить" : "Создать"}
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
          <Text size="sm" c="dimmed">
            Введите email пользователя, которого хотите добавить в группу
          </Text>
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
              loading={addingMember}
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
