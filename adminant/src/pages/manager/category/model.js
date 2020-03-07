import { createRule, listRule,  updateRule } from './service';


const Model = {
  namespace: 'managerCategoryModel',
  state: {
    listData: {
      list: [],
      pagination: {},
    },
  },
  effects: {
    *list({ payload }, { call, put }) {
      const response = yield call(listRule, payload);
      yield put({
        type: '_list',
        payload: response.data,
      });
    },

    *create({ payload, callback }, { call, put }) {
      yield call(createRule, payload);
      if (callback) callback();
      const response = yield call(listRule, payload);
      yield put({
        type: '_list',
        payload: response.data,
      });
    },

    *update({ payload, callback }, { call, put }) {
      yield call(updateRule, payload);
      if (callback) callback();
      const response = yield call(listRule, payload);
      yield put({
        type: '_list',
        payload: response.data,
      });
    },


    //搜索
    *search({ payload, callback }, { call, put }) {
      yield call(searchUsers, payload);
      if (callback) callback();
      const response = yield call(listRule, payload);
      yield put({
        type: '_list',
        payload: response.data,
      });
    },


    //用户管理列表
    *userList({ payload, callback }, { call, put }) {
      yield call(usersList, payload);
      if (callback) callback();
      const response = yield call(listRule, payload);
      yield put({
        type: '_list',
        payload: response.data,
      });
    },

    //禁用
    *enable({ payload, callback }, { call, put }) {
      yield call(enableUser, payload);
      if (callback) callback();
      const response = yield call(listRule, payload);
      yield put({
        type: '_list',
        payload: response.data,
      });
    },

    //删除
    *delete({ payload, callback }, { call, put }) {
      yield call(deleteUser, payload);
      if (callback) callback();
      const response = yield call(listRule, payload);
      yield put({
        type: '_list',
        payload: response.data,
      });
    },

  },
  reducers: {
    _list(state, action) {
      return {
        ...state,
        listData: action.payload
      };
    },
  },
};
export default Model;
