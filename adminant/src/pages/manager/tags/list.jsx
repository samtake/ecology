import {
    Button,
    Card,
    Col,
    Form,
    Input,
    Row,
    Select,
    message,
    Avatar,
  } from 'antd';
  import React, { Component, Fragment } from 'react';
  import { PageHeaderWrapper } from '@ant-design/pro-layout';
  import { connect } from 'dva';
  import moment from 'moment';
  import styles from './style.less';
//   import CreateForm from './components/CreateForm';
  import StandardTable from '@/components/SaberStandardTable';
//   import UpdateForm from './components/UpdateForm';

  const FormItem = Form.Item;
  const { Option } = Select;
  
  const getValue = obj =>
    Object.keys(obj)
      .map(key => obj[key])
      .join(',');
  
  
  /* eslint react/no-multi-comp:0 */
  @connect(({ toolModel, loading }) => ({
    listData:toolModel.listData,
    loading: loading.models.toolModel,
  }))
  class TableList extends Component {
    state = {
      modalVisible: false,
      updateModalVisible: false,
      expandForm: false,
      formValues: {},
      updateInitialValues: {},
    };
    columns = [
      {
        title: 'ID',
        dataIndex: 'id',
      },
      {
        title: '头像',
        dataIndex: 'avatar',
        render: (url) => (
          <div>
            <Avatar size={64} icon={<UserOutlined />} />
          </div>
        ),
      },
      {
        title: '账号',
        dataIndex: 'account',
      },
      {
        title: '角色',
        dataIndex: 'role',
      },
      {
        title: '状态',
        dataIndex: 'state',
      },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        sorter: true,
        render: val => <span>{moment(val).format('YYYY-MM-DD HH:mm:ss')}</span>,
      },
      {
        title: '操作',
        render: (text, record) => (
          <Fragment>
            <a  onClick={() => this.handleUpdateModalVisible(true, record)}>  编辑  </ a>
            <a style={{color:'red', margin:'0 auto'}} onClick={() => this.handleEnableModalVisible(true, record)}>  禁用  </a>
            <a style={{color:'red', margin:'0 auto'}} onClick={() => this.handleDeleteModalVisible(true, record)}>  删除  </a>
          </Fragment>
        ),
      },
    ];
  
    componentDidMount() {
      const { dispatch } = this.props;
      dispatch({
        type: 'managerTagsModel/list',
      });
    }
  
    handleStandardTableChange = (pagination, filtersArg, sorter) => {
      const { dispatch } = this.props;
      const { formValues } = this.state;
      const filters = Object.keys(filtersArg).reduce((obj, key) => {
        const newObj = { ...obj };
        newObj[key] = getValue(filtersArg[key]);
        return newObj;
      }, {});
      const params = {
        currentPage: pagination.current,
        pageSize: pagination.pageSize,
        ...formValues,
        ...filters,
      };
  
      if (sorter.field) {
        params.sorter = `${sorter.field}_${sorter.order}`;
      }
  
      dispatch({
        type: 'managerTagsModel/list',
        payload: params,
      });
    };
  
    handleFormReset = () => {
      const { form, dispatch } = this.props;
      form.resetFields();
      this.setState({
        formValues: {},
      });
      dispatch({
        type: 'managerTagsModel/list',
        payload: {},
      });
    };
  
    handleSearch = e => {
      e.preventDefault();
      const { dispatch, form } = this.props;
      form.validateFields((err, fieldsValue) => {
        if (err) return;
        const values = {
          ...fieldsValue,
          updatedAt: fieldsValue.updatedAt && fieldsValue.updatedAt.valueOf(),
        };
        this.setState({
          formValues: values,
        });
        dispatch({
          type: 'managerTagsModel/list',
          payload: values,
        });
      });
    };
  
  
    handleCreate = fields => {
      const { dispatch } = this.props;
      dispatch({
        type: 'managerTagsModel/create',
        payload: {
          ...fields,
        },
      });
      message.success('添加成功');
      this.handleModalVisible();
    };
  
    handleUpdate = fields => {
      const { dispatch } = this.props;
      dispatch({
        type: 'managerTagsModel/update',
        payload: {
          ...fields,
        },
      });
      message.success('配置成功');
      this.handleUpdateModalVisible();
    };
  
    handleModalVisible = flag => {
      this.setState({
        modalVisible: !!flag,
      });
    };
  
    //编辑
    handleUpdateModalVisible = fields => {
      const { dispatch } = this.props;
      dispatch({
        type: 'managerTagsModel/update',
        payload: {
          ...fields,
        },
      });
      message.success('编辑成功');
      this.handleUpdateModalVisible();
    };

    //禁用
    handleEnableModalVisible = fields => {
      const { dispatch } = this.props;
      dispatch({
        type: 'managerTagsModel/enable',
        payload: {
          ...fields,
        },
      });
      message.success('禁用成功');
      this.handleUpdateModalVisible();
    };


    //删除
    handleDeleteModalVisible = fields => {
      const { dispatch } = this.props;
      dispatch({
        type: 'managerTagsModel/delete',
        payload: {
          ...fields,
        },
      });
      message.success('删除成功');
      this.handleUpdateModalVisible();
    };
  
    handleSelectChange = (value) => {
        console.log(`selected ${value}`);
    };


  
    renderSimpleForm() {
      const { form } = this.props;
      const { getFieldDecorator } = form;
      return (
        <Form onSubmit={this.handleSearch} layout="inline">
          <Row
            gutter={{
              md: 8,
              lg: 24,
              xl: 48,
            }}
          >
            <Col md={6} sm={24}>
                <Select defaultValue="reader" style={{ width: 120 }} onChange={this.handleSelectChange}>
                    <Option value="all">全部</Option>
                    <Option value="manager">管理员</Option>
                    <Option value="author">作者</Option>
                    <Option value="reader">读者</Option>
                </Select>
            </Col>
            <Col md={6} sm={24}>
                <FormItem label="">
                    {getFieldDecorator('user')(<Input placeholder="搜索用户" />)}
                </FormItem>
            </Col>
            <Col md={6} sm={24}>
                <span className={styles.submitButtons}>
                    <Button type="primary" htmlType="submit">
                        提交
                    </Button>
                </span>
            </Col> 
          </Row>
        </Form>
      );
    }
  
    render() {
      const {
        listData,
        loading,
      } = this.props;
      const { modalVisible, updateModalVisible, updateInitialValues } = this.state;
  
      return (
        <PageHeaderWrapper>
          <Card bordered={false}>
            <div className={styles.tableList}>
              <div className={styles.tableListForm}>{this.renderSimpleForm()}</div>
              <div className={styles.tableListOperator}>
                <Button icon="plus" type="primary" onClick={() => this.handleModalVisible(true)}>
                  新建
                </Button>
              </div>
              <StandardTable
                loading={loading}
                data={listData}
                columns={this.columns}
                onChange={this.handleStandardTableChange}
              />
            </div>
          </Card>
          {/* <CreateForm
            handleCreate={this.handleCreate}
            handleModalVisible={this.handleModalVisible}
            modalVisible={modalVisible}
          />
          {updateInitialValues && Object.keys(updateInitialValues).length ? (
            <UpdateForm
              handleUpdateModalVisible={this.handleUpdateModalVisible}
              handleUpdate={this.handleUpdate}
              updateModalVisible={updateModalVisible}
              values={updateInitialValues}
            />
          ) : null} */}
        </PageHeaderWrapper>
      );
    }
  }
  
  export default Form.create()(TableList);
  