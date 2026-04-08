using Http;
using Rest;

namespace CallBooking;

@route("/api/my/groups")
interface Groups {
  // List user's fixed visibility groups (Family, Work, Friends)
  @get list(): GroupsListResponse | ErrorResponse;

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
