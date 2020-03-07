import request from '@/utils/request';

export async function listRule(params) {
  return request('/admin/tool/list', {
    params,
  });
}

export async function createRule(params) {
  return request('/admin/tool/create', {
    method: 'POST',
    data: { ...params },
  });
}
export async function updateRule(params) {
  return request('/admin/tool/update', {
    method: 'POST',
    data: { ...params },
  });
}



//搜索
export async function searchUsers(params) {
  return request('/admin/manager/userSearch', {
    method: 'GET',
    data: { ...params },
  });
}

//用户管理列表
export async function usersList(params) {
  return request('/admin/manager/users', {
    method: 'GET',
    data: { ...params },
  });
}


//禁用
export async function enableUser(params) {
  return request('/admin/manager/enable', {
    method: 'POST',
    data: { ...params },
  });
}

//删除
export async function deleteUser(params) {
  return request('/admin/manager/delete', {
    method: 'POST',
    data: { ...params },
  });
}


