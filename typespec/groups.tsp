using Http;
using Rest;

namespace CallBooking;

@route("/api/my/groups")
interface Groups {
  @get list(): GroupsListResponse | ErrorResponse;

  @post create(@body group: CreateGroupRequest): { @statusCode _: 201; ...VisibilityGroup } | ErrorResponse;

  @put
  @route("/{id}")
  update(@path id: string, @body group: CreateGroupRequest): VisibilityGroup | ErrorResponse;

  @delete
  @route("/{id}")
  delete(@path id: string): { @statusCode _: 204; } | ErrorResponse;

  @get
  @route("/{id}/members")
  listMembers(@path id: string): MembersListResponse | ErrorResponse;

  @post
  @route("/{id}/members")
  addMember(@path id: string, @body req: AddMemberRequest): { @statusCode _: 201; ...GroupMember } | ErrorResponse;

  @delete
  @route("/{id}/members/{memberId}")
  removeMember(@path id: string, @path memberId: string): { @statusCode _: 204; } | ErrorResponse;
}
