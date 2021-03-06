/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package inst

import (
	"context"
	"io"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/condition"
	frtypes "configcenter/src/common/mapstr"
	metatype "configcenter/src/common/metadata"
)

func (cli *inst) updateMainlineAssociation(child Inst, parentID int64) error {

	childID, err := child.GetInstID()
	if nil != err {
		return err
	}

	cond := condition.CreateCondition()
	cond.Field(child.GetObject().GetInstIDFieldName()).Eq(int(childID))
	if child.GetObject().IsCommon() {
		cond.Field(metatype.ModelFieldObjectID).Eq(child.GetObject().GetID())
	}

	data := frtypes.MapStr{}
	data.Set("data", frtypes.MapStr{
		common.BKInstParentStr: parentID,
	})
	data.Set("condition", cond.ToMapStr())

	rsp, err := cli.clientSet.ObjectController().Instance().UpdateObject(context.Background(), child.GetObject().GetObjectType(), cli.params.Header, data)
	if nil != err {
		blog.Errorf("[inst-inst] failed to request object controller, error info %s", err.Error())
		return cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if common.CCSuccess != rsp.Code {
		blog.Errorf("[inst-inst] failed to update the association, error info is %s", rsp.ErrMsg)
		return cli.params.Err.Error(rsp.Code)
	}

	return nil
}

func (cli *inst) setCommonInstAssociation(child Inst, parent Inst) error {

	parentID, err := parent.GetInstID()
	if nil != err {
		return err
	}

	childID, err := child.GetInstID()
	if nil != err {
		return err
	}

	asstItems, err := cli.searchInstAssociation(childID, parentID, child.GetObject().GetID(), parent.GetObject().GetID())
	if nil != err {
		return err
	}

	// construct the association
	asst := metatype.InstAsst{}
	asst.AsstInstID = parentID
	asst.InstID = childID
	asst.ObjectID = child.GetObject().GetID()
	asst.AsstObjectID = parent.GetObject().GetID()

	// create a new association
	if 0 != len(asstItems) {

		rsp, err := cli.clientSet.ObjectController().Instance().CreateObject(context.Background(), common.BKTableNameInstAsst, cli.params.Header, asst.ToMapStr())
		if nil != err {
			blog.Errorf("[inst-asst] failed to request the object controller,error info is %s", err.Error())
			return cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
		}

		if common.CCSuccess != rsp.Code {
			blog.Errorf("[inst-asst] failed to create the common inst association, error info is %s", rsp.ErrMsg)
			return cli.params.Err.Error(rsp.Code)
		}

		return nil
	}

	// update the association
	for _, item := range asstItems {

		originAsst := metatype.InstAsst{}
		if _, err = originAsst.Parse(item); nil != err {
			blog.Errorf("[inst-asst] failed to parse the inst asst data(%#v), error info is %s", item, err.Error())
			return err
		}

		cond := condition.CreateCondition()
		cond.Field("id").Eq(originAsst.ID)

		data := frtypes.MapStr{}
		data.Set("data", asst.ToMapStr())
		data.Set("condition", cond.ToMapStr())

		rsp, err := cli.clientSet.ObjectController().Instance().UpdateObject(context.Background(), common.BKTableNameInstAsst, cli.params.Header, data)
		if nil != err {
			blog.Errorf("[inst-asst] failed to request object controller, error info %s", err.Error())
			return cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
		}

		if common.CCSuccess != rsp.Code {
			blog.Errorf("[inst-asst] failed to update the association, error info is %s", rsp.ErrMsg)
			return cli.params.Err.Error(rsp.Code)
		}
	}

	return nil
}

func (cli *inst) searchInstAssociation(instID, asstInstID int64, objID, asstObjID string) ([]frtypes.MapStr, error) {

	cond := condition.CreateCondition()

	if 0 < instID {
		cond.Field(common.BKInstIDField).Eq(instID)
	}
	if 0 < asstInstID {
		cond.Field(common.BKAsstInstIDField).Eq(asstInstID)
	}
	if 0 != len(objID) {
		cond.Field(common.BKObjIDField).Eq(objID)
	}
	if 0 != len(asstObjID) {
		cond.Field(common.BKAsstObjIDField).Eq(asstObjID)
	}

	queryInput := &metatype.QueryInput{}
	queryInput.Condition = cond.ToMapStr()
	queryInput.Limit = common.BKNoLimit
	rsp, err := cli.clientSet.ObjectController().Instance().SearchObjects(context.Background(), common.BKTableNameInstAsst, cli.params.Header, queryInput)
	if nil != err {
		blog.Errorf("[inst-inst] failed to request the object controller , error info is %s", err.Error())
		return nil, cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if common.CCSuccess != rsp.Code {
		blog.Errorf("[inst-inst] failed to search the inst association, error info is %s", rsp.ErrMsg)
		return nil, cli.params.Err.Error(rsp.Code)
	}

	return rsp.Data.Info, nil

}

func (cli *inst) deleteInstAssociation(instID, asstInstID int64, objID, asstObjID string) error {

	cond := condition.CreateCondition()

	cond.Field(common.BKInstIDField).Eq(instID)
	cond.Field(common.BKAsstInstIDField).Eq(asstInstID)
	cond.Field(common.BKObjIDField).Eq(objID)
	cond.Field(common.BKAsstObjIDField).Eq(asstObjID)

	rsp, err := cli.clientSet.ObjectController().Instance().DelObject(context.Background(), common.BKTableNameInstAsst, cli.params.Header, cond.ToMapStr())
	if nil != err {
		blog.Errorf("[inst-inst] failed to request the object controller , error info is %s", err.Error())
		return cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if common.CCSuccess != rsp.Code {
		blog.Errorf("[inst-inst] failed to delete the inst association, error info is %s", rsp.ErrMsg)
		return cli.params.Err.Error(rsp.Code)
	}

	return nil

}

func (cli *inst) GetMainlineParentInst() (Inst, error) {

	parentObj, err := cli.target.GetMainlineParentObject()
	if nil != err {
		return nil, err
	}

	parentID, err := cli.GetParentID()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the inst id, error info is %s", err.Error())
		return nil, err
	}

	cond := condition.CreateCondition()
	cond.Field(metatype.ModelFieldOwnerID).Eq(cli.params.SupplierAccount)
	if parentObj.IsCommon() {
		cond.Field(metatype.ModelFieldObjectID).Eq(parentObj.GetID())
	}
	cond.Field(parentObj.GetInstIDFieldName()).Eq(parentID)

	rspItems, err := cli.searchInsts(parentObj, cond)
	if nil != err {
		blog.Errorf("[inst-inst] failed to request the object controller , error info is %s", err.Error())
		return nil, cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	for _, item := range rspItems {
		return item, nil // only one mainline parent
	}

	return nil, io.EOF
}
func (cli *inst) GetMainlineChildInst() ([]Inst, error) {

	childObj, err := cli.target.GetMainlineChildObject()
	if nil != err {
		if err == io.EOF {
			return []Inst{}, nil
		}
		blog.Errorf("[inst-inst]failed to get the object(%s)'s child object, error info is %s", cli.target.GetID(), err.Error())
		return nil, err
	}

	currInstID, err := cli.GetInstID()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the inst id, error info is %s", err.Error())
		return nil, err
	}

	cond := condition.CreateCondition()
	cond.Field(metatype.ModelFieldOwnerID).Eq(cli.params.SupplierAccount)
	if childObj.IsCommon() {
		cond.Field(metatype.ModelFieldObjectID).Eq(childObj.GetID())
	} else if childObj.GetID() == common.BKInnerObjIDSet {
		cond.Field(common.BKDefaultField).NotEq(common.DefaultResSetFlag)
	}
	cond.Field(common.BKInstParentStr).Eq(currInstID)
	return cli.searchInsts(childObj, cond)
}
func (cli *inst) GetParentObjectWithInsts() ([]*ObjectWithInsts, error) {

	result := make([]*ObjectWithInsts, 0)
	parentObjs, err := cli.target.GetParentObject()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the object(%s)'s parent, error info is %s", cli.target.GetID(), err.Error())
		return result, err
	}

	currInstID, err := cli.GetInstID()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the inst id, error info is %s", err.Error())
		return result, err
	}

	for _, parentObj := range parentObjs {

		rstObj := &ObjectWithInsts{Object: parentObj}
		asstItems, err := cli.searchInstAssociation(-1, currInstID, parentObj.GetID(), cli.target.GetID())
		if nil != err {
			blog.Errorf("[inst-inst] failed to search the inst association, the error info is %s", err.Error())
			return result, err
		}

		parentInstIDS := []int64{}
		for _, item := range asstItems {

			parentInstID, err := item.Int64(common.BKInstIDField)
			if nil != err {
				blog.Errorf("[inst-inst] failed to parse the asst inst id, error info is %s", err.Error())
				return result, err
			}
			parentInstIDS = append(parentInstIDS, parentInstID)
		}

		innerCond := condition.CreateCondition()

		innerCond.Field(metatype.ModelFieldOwnerID).Eq(cli.params.SupplierAccount)
		innerCond.Field(parentObj.GetInstIDFieldName()).In(parentInstIDS)
		if parentObj.IsCommon() {
			innerCond.Field(metatype.ModelFieldObjectID).Eq(parentObj.GetID())
		}

		rspItems, err := cli.searchInsts(parentObj, innerCond)
		if nil != err {
			blog.Errorf("[inst-inst] failed to search the insts by the condition(%#v), error info is %s", innerCond, err.Error())
			return result, err
		}

		rstObj.Insts = rspItems
		result = append(result, rstObj)

	}

	return result, nil
}

func (cli *inst) GetParentInst() ([]Inst, error) {

	parentObjs, err := cli.target.GetParentObject()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the object(%s)'s parent, error info is %s", cli.target.GetID(), err.Error())
		return nil, err
	}

	currInstID, err := cli.GetInstID()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the inst id, error info is %s", err.Error())
		return nil, err
	}

	result := make([]Inst, 0)

	for _, parentObj := range parentObjs {
		asstItems, err := cli.searchInstAssociation(-1, currInstID, parentObj.GetID(), cli.target.GetID())
		if nil != err {
			blog.Errorf("[inst-inst] failed to search the inst association, the error info is %s", err.Error())
			return nil, err
		}

		parentInstIDS := []int64{}
		for _, item := range asstItems {

			parentInstID, err := item.Int64(common.BKInstIDField)
			if nil != err {
				blog.Errorf("[inst-inst] failed to parse the asst inst id, error info is %s", err.Error())
				return nil, err
			}

			parentInstIDS = append(parentInstIDS, parentInstID)
		}

		innerCond := condition.CreateCondition()
		innerCond.Field(metatype.ModelFieldOwnerID).Eq(cli.params.SupplierAccount)
		innerCond.Field(parentObj.GetInstIDFieldName()).In(parentInstIDS)
		if parentObj.IsCommon() {
			innerCond.Field(metatype.ModelFieldObjectID).Eq(parentObj.GetID())
		}

		rspItems, err := cli.searchInsts(parentObj, innerCond)
		if nil != err {
			blog.Errorf("[inst-asst] failed to search the insts by the condition(%#v), error info is %s", innerCond, err.Error())
			return nil, err
		}

		result = append(result, rspItems...)

	}

	return result, nil
}

func (cli *inst) GetChildObjectWithInsts() ([]*ObjectWithInsts, error) {

	result := make([]*ObjectWithInsts, 0)

	childObjs, err := cli.target.GetChildObject()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the object(%s)'s child, error info is %s", cli.target.GetID(), err.Error())
		return result, err
	}

	currInstID, err := cli.GetInstID()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the inst id, error info is %s", err.Error())
		return result, err
	}

	for _, childObj := range childObjs {

		rstObj := &ObjectWithInsts{Object: childObj}
		asstItems, err := cli.searchInstAssociation(currInstID, -1, cli.target.GetID(), childObj.GetID())
		if nil != err {
			blog.Errorf("[inst-inst] failed to search the inst association,  the error info is %s", err.Error())
			return result, err
		}

		childInstIDS := make([]int64, 0)
		for _, item := range asstItems {
			childInstID, err := item.Int64(common.BKAsstInstIDField)
			if nil != err {
				blog.Errorf("[inst-inst] failed to parse the asst inst id, error info is %s", err.Error())
				return result, err
			}
			childInstIDS = append(childInstIDS, childInstID)
		}

		innerCond := condition.CreateCondition()
		innerCond.Field(metatype.ModelFieldOwnerID).Eq(cli.params.SupplierAccount)
		innerCond.Field(childObj.GetInstIDFieldName()).In(childInstIDS)
		if childObj.IsCommon() {
			innerCond.Field(metatype.ModelFieldObjectID).Eq(childObj.GetID())
		}

		rspItems, err := cli.searchInsts(childObj, innerCond)
		if nil != err {
			blog.Errorf("[inst-inst] failed to search the insts by the condition(%#v), error info is %s", innerCond, err.Error())
			return result, err
		}

		rstObj.Insts = rspItems
		result = append(result, rstObj)
	}

	return result, nil
}
func (cli *inst) GetChildInst() ([]Inst, error) {

	childObjs, err := cli.target.GetChildObject()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the object(%s)'s child, error info is %s", cli.target.GetID(), err.Error())
		return nil, err
	}

	currInstID, err := cli.GetInstID()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the inst id, error info is %s", err.Error())
		return nil, err
	}

	result := make([]Inst, 0)

	for _, childObj := range childObjs {

		asstItems, err := cli.searchInstAssociation(currInstID, -1, cli.target.GetID(), childObj.GetID())
		if nil != err {
			blog.Errorf("[inst-inst] failed to search the inst association,  the error info is %s", err.Error())
			return nil, err
		}

		childInstIDS := []int64{}
		for _, item := range asstItems {

			childInstID, err := item.Int64(common.BKInstIDField)
			if nil != err {
				blog.Errorf("[inst-inst] failed to parse the asst inst id, error info is %s", err.Error())
				return nil, err
			}
			childInstIDS = append(childInstIDS, childInstID)
		}

		innerCond := condition.CreateCondition()
		innerCond.Field(metatype.ModelFieldOwnerID).Eq(cli.params.SupplierAccount)
		innerCond.Field(childObj.GetInstIDFieldName()).In(childInstIDS)
		if childObj.IsCommon() {
			innerCond.Field(metatype.ModelFieldObjectID).Eq(childObj.GetID())
		}

		rspItems, err := cli.searchInsts(childObj, innerCond)
		if nil != err {
			blog.Errorf("[inst-inst] failed to search the insts by the condition(%#v), error info is %s", innerCond, err.Error())
			return nil, err
		}

		result = append(result, rspItems...)
	}

	return result, nil
}

func (cli *inst) SetMainlineParentInst(instID int64) error {
	if err := cli.updateMainlineAssociation(cli, instID); nil != err {
		blog.Errorf("[inst-inst] failed to update the mainline association, error info is %s", err.Error())
		return err
	}

	return nil
}
func (cli *inst) SetMainlineChildInst(targetInst Inst) error {

	instID, err := targetInst.GetInstID()
	if err != nil {
		return err
	}

	childInsts, err := cli.GetMainlineChildInst()
	if nil != err {
		blog.Errorf("[inst-inst] failed to get the child inst, error info is  %s", err.Error())
		return err
	}
	for _, childInst := range childInsts {
		if err = cli.updateMainlineAssociation(childInst, instID); nil != err {
			blog.Errorf("[inst-inst] failed to set the mainline child inst, error info is %s", err.Error())
			return err
		}
	}

	id, err := cli.GetInstID()
	if err != nil {
		return err
	}

	if err = cli.updateMainlineAssociation(targetInst, id); nil != err {
		blog.Errorf("[inst-inst] failed to update the mainline association, error info is %s", err.Error())
		return err
	}

	return nil
}

func (cli *inst) SetParentInst(targetInst Inst) error {
	return cli.setCommonInstAssociation(cli, targetInst)
}
func (cli *inst) SetChildInst(targetInst Inst) error {
	return cli.setCommonInstAssociation(targetInst, cli)
}
